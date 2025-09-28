package proxy

import (
	"context"
	stdmath "math"
	"math/big"
	"sync"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayMeter = (*ProxyRelayMeter)(nil)

// sessionRelayMeter is the relay meter's internal representation of an onchain
// Application's max and consumed stake.
type sessionRelayMeter struct {
	// The onchain application the relay meter is for.
	app apptypes.Application
	// The maximum uPOKT an application can pay this relayer for a given session.
	// This is a fraction of the Application's overall stake in proportion.
	maxCoin cosmostypes.Coin
	// The amount of uPOKT a specific application has consumed from this relayer in the given session.
	consumedCoin cosmostypes.Coin
	// The header for the session the Application and Supplier (backed by the relayer)
	// are exchanging services in.
	sessionHeader *sessiontypes.SessionHeader
	// numOverServicedRelays is the number of relays that have been over-serviced
	// by the relayer for the application.
	numOverServicedRelays uint64
}

// ProxyRelayMeter is the offchain Supplier's rate limiter.
// It ensures that no Application is over-serviced by the Supplier per session.
// This is done by maintaining the max amount of stake the supplier can consume
// per session and the amount of stake consumed by mined relays.
// TODO_POST_MAINNET(@red-0ne): Consider making the relay meter a light client,
// since it's already receiving all committed blocks and events.
type ProxyRelayMeter struct {
	// sessionToRelayMeterMap is a map of session IDs to their corresponding session relay meter.
	// Only known applications (i.e. have sent at least one relay) have their stakes metered.
	// This map gets reset every new session in order to meter new applications.
	sessionToRelayMeterMap map[string]*sessionRelayMeter

	// overServicingEnabled allows Suppliers to overservice applications.
	// This entails providing a free service (i.e. mine for relays), that they will not be paid for onchain.
	// This is common by some suppliers to build goodwill and receive a higher offchain quality-of-service rating.
	overServicingEnabled bool

	// relayMeterMu ensures that relay meter operations are thread-safe.
	relayMeterMu sync.RWMutex

	// Clients to query onchain data.
	applicationQuerier client.ApplicationQueryClient
	serviceQuerier     client.ServiceQueryClient
	sharedQuerier      client.SharedQueryClient
	sessionQuerier     client.SessionQueryClient
	blockQuerier       client.BlockClient

	logger polylog.Logger
}

func NewRelayMeter(deps depinject.Config, enableOverServicing bool) (relayer.RelayMeter, error) {
	rm := &ProxyRelayMeter{
		sessionToRelayMeterMap: make(map[string]*sessionRelayMeter),
		overServicingEnabled:   enableOverServicing,
	}

	if err := depinject.Inject(
		deps,
		&rm.sharedQuerier,
		&rm.applicationQuerier,
		&rm.serviceQuerier,
		&rm.blockQuerier,
		&rm.sessionQuerier,
		&rm.logger,
	); err != nil {
		return nil, err
	}

	return rm, nil
}

// Start starts the relay meter by observing application staked events and new sessions.
func (rmtr *ProxyRelayMeter) Start(ctx context.Context) error {
	// Listen to new blocks and reset the relay meter application stakes every new session.
	committedBlocksSequence := rmtr.blockQuerier.CommittedBlocksSequence(ctx)
	channel.ForEach(ctx, committedBlocksSequence, rmtr.forEachNewBlockFn)

	return nil
}

// IsOverServicing returns whether the relay would result in over-servicing the application.
//
// It returns true if serving this relay would exceed the application's allocated stake
// (serving beyond what the application can pay for), false if the relay is within limits.
// The function updates the relay meter with the given relay request metadata.
func (rmtr *ProxyRelayMeter) IsOverServicing(
	ctx context.Context,
	reqMeta servicetypes.RelayRequestMetadata,
) bool {
	// Create a context-specific logger to avoid concurrent access issues
	logger := rmtr.logger.With(
		"method", "IsOverServicing",
		"session_id", reqMeta.GetSessionHeader().GetSessionId(),
	)

	// Ensure that the served application has a relay meter and update the consumed
	// stake amount.
	appRelayMeter, err := rmtr.ensureRequestSessionRelayMeter(ctx, reqMeta)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to set up relay meter in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return false
	}

	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to set up relay meter in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return false
	}

	service, err := rmtr.serviceQuerier.GetService(ctx, reqMeta.SessionHeader.ServiceId)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to set up relay meter in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return false
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCostCoin, err := getSingleRelayCostCoin(sharedParams, &service)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to calculate relay cost in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return false
	}

	// Increase the consumed stake amount by relay cost.
	newConsumedCoin := appRelayMeter.consumedCoin.Add(relayCostCoin)

	if appRelayMeter.maxCoin.IsGTE(newConsumedCoin) {
		appRelayMeter.consumedCoin = newConsumedCoin
		return false
	}

	appRelayMeter.numOverServicedRelays++

	// Exponential backoff, only log over-servicing when numOverServicedRelays is a power of 2
	if shouldLogOverServicing(appRelayMeter.numOverServicedRelays) {
		logger.Warn().Msgf(
			"overservicing enabled, application %q over-serviced %s",
			appRelayMeter.app.GetAddress(),
			appRelayMeter.numOverServicedRelays,
		)
	}

	return true
}

// SetNonApplicableRelayReward updates the relay meter to make the relay reward for
// the given relay request as non-applicable.
// This is used when the relay is not volume / reward applicable but was optimistically
// accounted for in the relay meter.
func (rmtr *ProxyRelayMeter) SetNonApplicableRelayReward(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	// Create a context-specific logger to avoid concurrent access issues
	logger := rmtr.logger.With(
		"method", "SetNonApplicableRelayReward",
		"session_id", reqMeta.GetSessionHeader().GetSessionId(),
	)

	sessionRelayMeter, ok := rmtr.sessionToRelayMeterMap[reqMeta.GetSessionHeader().GetSessionId()]
	if !ok {
		logger.Warn().Msgf(
			"[Non critical] Unable to find session relay meter for session %s. Application may be rate limited more than intended: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			ErrRelayerProxyUnknownSession.Wrap("session relay meter not found"),
		)

		return
	}

	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to set up relay meter in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return
	}

	service, err := rmtr.serviceQuerier.GetService(ctx, reqMeta.SessionHeader.ServiceId)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to set up relay meter in session %s. Relay will continue without rate limiting: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
		return
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCost, err := getSingleRelayCostCoin(sharedParams, &service)
	if err != nil {
		logger.Warn().Msgf(
			"[Non critical] Unable to calculate relay cost in session %s. Application may be rate limited more than intended: %v",
			reqMeta.GetSessionHeader().GetSessionId(),
			err,
		)
	}

	// TODO_FOLLOWUP(@red-0ne): Consider fixing the relay meter logic to never have
	// a less than relay cost consumed amount.
	if sessionRelayMeter.consumedCoin.IsLT(relayCost) {
		logger.Warn().Msgf(
			"(SHOULD NEVER HAPPEN) Your session earned less than the cost of a single relay. Not submitting a claim for application (%s), service id: (%s), session id: (%s), with consumed amount: (%s), relay cost: (%s)",
			sessionRelayMeter.app.GetAddress(),
			sessionRelayMeter.sessionHeader.GetServiceId(),
			sessionRelayMeter.sessionHeader.GetSessionId(),
			sessionRelayMeter.consumedCoin.String(),
			relayCost.String(),
		)
		return
	}
	// Decrease the consumed stake amount by relay cost.
	newConsumedAmount := sessionRelayMeter.consumedCoin.Sub(relayCost)

	sessionRelayMeter.consumedCoin = newConsumedAmount
}

// AllowOverServicing returns true if the relay meter is configured to allow over-servicing.
//
// Over-servicing allows the offchain relay miner to serve more relays than the
// amount of stake the onchain Application can pay the corresponding onchain
// Supplier at the end of the session.
func (rmtr *ProxyRelayMeter) AllowOverServicing() bool {
	// Over-servicing is enabled if the relay meter is configured to allow it.
	return rmtr.overServicingEnabled
}

// forEachNewBlockFn is a callback function that is called every time a new block is committed.
// It resets the relay meter's application stakes every new session so that new
// application stakes can be metered.
func (rmtr *ProxyRelayMeter) forEachNewBlockFn(ctx context.Context, block client.Block) {
	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		return
	}

	// First pass: Read-only scan to identify sessions to delete
	rmtr.relayMeterMu.RLock()
	var sessionsToDelete []string
	for sessionId, sessionRelayMeter := range rmtr.sessionToRelayMeterMap {
		sessionClaimOpenHeight := sharedtypes.GetClaimWindowOpenHeight(
			sharedParams,
			sessionRelayMeter.sessionHeader.GetSessionEndBlockHeight(),
		)

		if block.Height() >= sessionClaimOpenHeight {
			sessionsToDelete = append(sessionsToDelete, sessionId)
		}
	}
	rmtr.relayMeterMu.RUnlock()

	// Second pass: Write lock only for deletions (if needed)
	if len(sessionsToDelete) > 0 {
		return
	}

	// DEV_NOTE: This is a WRITE lock on relay-meter-wide mutex, shared across all served relays:
	// - This critical section is delicate for performance.
	// - Keep the work here minimal and fast.
	// - Avoid unnecessary allocations, I/O, or blocking calls.
	// - Prolonged holds will throttle relay throughput across the process.
	rmtr.relayMeterMu.Lock()
	for _, sessionId := range sessionsToDelete {
		// The session started its claim phase and the corresponding session relay meter
		// is no longer needed.
		delete(rmtr.sessionToRelayMeterMap, sessionId)
	}
	rmtr.relayMeterMu.Unlock()
}

// ensureRequestSessionRelayMeter ensures that the relay miner has a relay meter
// ready for monitoring the requests's application's consumption.
func (rmtr *ProxyRelayMeter) ensureRequestSessionRelayMeter(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) (*sessionRelayMeter, error) {
	appAddress := reqMeta.GetSessionHeader().GetApplicationAddress()
	sessionId := reqMeta.GetSessionHeader().GetSessionId()

	rmtr.relayMeterMu.RLock()
	relayMeter, ok := rmtr.sessionToRelayMeterMap[sessionId]
	rmtr.relayMeterMu.RUnlock()
	// If the application is seen for the first time in this session, calculate the
	// max amount of stake the application can consume.
	if !ok {
		var app apptypes.Application
		// Application stake is guaranteed to be up-to-date as long as the cache is
		// invalidated at each new block.
		app, err := rmtr.applicationQuerier.GetApplication(ctx, appAddress)
		if err != nil {
			return nil, err
		}

		// In order to prevent over-servicing, the protocol must split the application's stake
		// among all the suppliers that are serving it.
		if len(app.ServiceConfigs) != 1 {
			return nil, ErrRelayerProxyInvalidSession.Wrapf(
				"application %q has %d service configs, expected 1",
				appAddress,
				len(app.ServiceConfigs),
			)
		}

		sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		sessionParams, err := rmtr.sessionQuerier.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		// calculate the max amount of stake the application can consume in the current session.
		supplierAppStake := getAppStakePortionPayableToSessionSupplier(
			app.GetStake(),
			sharedParams,
			sessionParams.GetNumSuppliersPerSession(),
		)

		relayMeter = &sessionRelayMeter{
			app:           app,
			consumedCoin:  cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
			maxCoin:       supplierAppStake,
			sessionHeader: reqMeta.SessionHeader,
		}

		rmtr.relayMeterMu.Lock()
		rmtr.sessionToRelayMeterMap[sessionId] = relayMeter
		rmtr.relayMeterMu.Unlock()
	}

	return relayMeter, nil
}

// getSingleRelayCostCoin returns the cost of a relay based on the shared parameters and the service.
//
// relayCost =
//
//	Compute Units Per Relay (CUPR) *
//	Compute Units To Token Multiplier (CUTTM) /
//	Compute Unit Cost Granularity
//
// Example:
// 1 relayCost (in uPOKT) =
//
//	100 (compute units per relay)
//	42_000_000 (compute unit cost in pPOKT) /
//	1000000 (convert pPOKT to uPOKT)
func getSingleRelayCostCoin(
	sharedParams *sharedtypes.Params,
	service *sharedtypes.Service,
) (cosmostypes.Coin, error) {
	// Get the cost of a single compute unit in fractional uPOKT.
	computeUnitCostUpokt := new(big.Rat).SetFrac64(
		int64(sharedParams.GetComputeUnitsToTokensMultiplier()),
		int64(sharedParams.GetComputeUnitCostGranularity()),
	)
	// Get the cost of a single relay in fractional uPOKT.
	relayCostRat := new(big.Rat).Mul(new(big.Rat).SetUint64(service.ComputeUnitsPerRelay), computeUnitCostUpokt)

	// Get the estimated cost of the relay if it gets mined in uPOKT.
	estimatedRelayCost := big.NewInt(0).Quo(relayCostRat.Num(), relayCostRat.Denom())
	estimatedRelayCostCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewIntFromBigInt(estimatedRelayCost))

	return estimatedRelayCostCoin, nil
}

// getAppStakePortionPayableToSessionSupplier returns the portion of the application
// stake that can be consumed per supplier per session.
func getAppStakePortionPayableToSessionSupplier(
	stake *cosmostypes.Coin,
	sharedParams *sharedtypes.Params,
	numSuppliersPerSession uint64,
) cosmostypes.Coin {
	appStakePerSupplier := stake.Amount.Quo(math.NewInt(int64(numSuppliersPerSession)))

	// Calculate the number of pending sessions that might consume the application's stake.
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	numBlocksUntilProofWindowCloses := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	numClosedSessionsAwaitingSettlement := stdmath.Ceil(float64(numBlocksUntilProofWindowCloses) / float64(numBlocksPerSession))
	// Add 1 to the number of pending sessions to account for the current session
	pendingSessions := int64(numClosedSessionsAwaitingSettlement) + 1

	appStakePerSessionSupplier := appStakePerSupplier.Quo(math.NewInt(pendingSessions))
	appStakePerSessionSupplierCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, appStakePerSessionSupplier)

	return appStakePerSessionSupplierCoin
}

// shouldLogOverServicing returns true if the number of occurrences is a power of 2.
// This is used to log the over-servicing warning with an exponential backoff.
func shouldLogOverServicing(occurrence uint64) bool {
	return (occurrence & (occurrence - 1)) == 0
}
