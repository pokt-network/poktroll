package proxy

import (
	"context"
	"math/big"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
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
	numOverServicedRelays int

	// sharedParams, service and serviceRelayDifficulty are used to calculate the relay cost
	// that increments the consumedAmount.
	// They are cached at each session to avoid querying the blockchain for each relay.
	// TODO_TECHDEBT(#543): Remove once the query clients start handling caching and invalidation.
	sharedParams           *sharedtypes.Params
	service                *sharedtypes.Service
	serviceRelayDifficulty servicetypes.RelayMiningDifficulty
}

// ProxyRelayMeter is the offchain Supplier's rate limiter.
// It ensures that no Application is over-serviced by the Supplier per session.
// This is done by maintaining the max amount of stake the supplier can consume
// per session and the amount of stake consumed by mined relays.
type ProxyRelayMeter struct {
	// sessionToRelayMeterMap is a map of session IDs to their corresponding session relay meter.
	// Only known applications (i.e. have sent at least one relay) have their stakes metered.
	// This map gets reset every new session in order to meter new applications.
	sessionToRelayMeterMap map[string]*sessionRelayMeter
	// overServicingAllowanceCoins allows Suppliers to overservice applications.
	// This entails providing a free service (i.e. mine for relays), that they will not be paid for onchain.
	// This is common by some suppliers to build goodwill and receive a higher offchain quality-of-service rating.
	// If negative, allow infinite overservicing.
	// TODO_MAINNET(@red-0ne): Expose overServicingAllowanceCoins as a configuration parameter.
	overServicingAllowanceCoins cosmostypes.Coin

	// relayMeterMu ensures that relay meter operations are thread-safe.
	relayMeterMu sync.Mutex

	// Clients to query onchain data.
	applicationQuerier client.ApplicationQueryClient
	serviceQuerier     client.ServiceQueryClient
	sharedQuerier      client.SharedQueryClient
	eventsQueryClient  client.EventsQueryClient
	blockQuerier       client.BlockClient

	logger polylog.Logger
}

func NewRelayMeter(deps depinject.Config) (relayer.RelayMeter, error) {
	overservicingAllowanceCoins := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000)
	rm := &ProxyRelayMeter{
		sessionToRelayMeterMap:      make(map[string]*sessionRelayMeter),
		overServicingAllowanceCoins: overservicingAllowanceCoins,
	}

	if err := depinject.Inject(
		deps,
		&rm.sharedQuerier,
		&rm.applicationQuerier,
		&rm.serviceQuerier,
		&rm.blockQuerier,
		&rm.eventsQueryClient,
		&rm.logger,
	); err != nil {
		return nil, err
	}

	return rm, nil
}

// Start starts the relay meter by observing application staked events and new sessions.
func (rmtr *ProxyRelayMeter) Start(ctx context.Context) error {
	// Listen to transaction events to filter application staked events.
	// TODO_BETA(@red-0ne): refactor this listener to be shared across all query clients
	// and remove the need to listen to events in the relay meter.
	eventsObs, err := rmtr.eventsQueryClient.EventsBytes(ctx, "tm.event = 'Tx'")
	if err != nil {
		return err
	}

	// Listen for application staked events and update known application stakes.
	//
	// Since an applications might upstake (never downstake) during a session, this
	// stake increase is guaranteed to be available at settlement.
	// Stake updates take effect immediately.
	//
	// This enables applications to adjust their stake mid-session and increase
	// their rate limits without needing to wait for the next session to start.
	appStakedEvents := filterTypedEvents[*apptypes.EventApplicationStaked](ctx, eventsObs, nil)
	channel.ForEach(ctx, appStakedEvents, rmtr.forEachEventApplicationStakedFn)

	// Listen to new blocks and reset the relay meter application stakes every new session.
	committedBlocksSequence := rmtr.blockQuerier.CommittedBlocksSequence(ctx)
	channel.ForEach(ctx, committedBlocksSequence, rmtr.forEachNewBlockFn)

	return nil
}

// AccumulateRelayReward accumulates the relay reward for the given relay request.
// The relay reward is added optimistically, assuming that the relay will be volume / reward
// applicable and the relay meter would remain up to date.
func (rmtr *ProxyRelayMeter) AccumulateRelayReward(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) error {
	// TODO_MAINNET: Locking the relay serving flow to ensure that the relay meter is updated
	// might be a bottleneck since ensureRequestAppMetrics is performing multiple
	// sequential queries to the Pocket Network node.
	// Re-evaluate when caching and invalidation is implemented.
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	// Ensure that the served application has a relay meter and update the consumed
	// stake amount.
	appRelayMeter, err := rmtr.ensureRequestSessionRelayMeter(ctx, reqMeta)
	if err != nil {
		return err
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCostCoin, err := getSingleMinedRelayCostCoin(
		appRelayMeter.sharedParams,
		appRelayMeter.service,
		appRelayMeter.serviceRelayDifficulty,
	)
	if err != nil {
		return err
	}

	// Increase the consumed stake amount by relay cost.
	newConsumedCoin := appRelayMeter.consumedCoin.Add(relayCostCoin)

	isAppOverServiced := appRelayMeter.maxCoin.IsLT(newConsumedCoin)

	if !isAppOverServiced {
		appRelayMeter.consumedCoin = newConsumedCoin
		return nil
	}

	// Check if the supplier is allowing unlimited over-servicing (i.e. negative value)
	allowUnlimitedOverServicing := rmtr.overServicingAllowanceCoins.IsNegative()

	// The application is over-servicing, if unlimited over-servicing is not allowed
	// and the newConsumedCoin is greater than the maxCoin + overServicingAllowanceCoins,
	// then return a rate limit error.
	overServicingCoin := newConsumedCoin.Sub(appRelayMeter.maxCoin)

	// In case Allowance is positive, add it to the maxCoin to allow no or limited over-servicing.
	if !allowUnlimitedOverServicing {
		maxAllowedOverServicing := appRelayMeter.maxCoin.Add(rmtr.overServicingAllowanceCoins)
		if maxAllowedOverServicing.IsLT(newConsumedCoin) {
			return ErrRelayerProxyRateLimited.Wrapf(
				"application has been rate limited, stake needed: %s, has: %s, ",
				newConsumedCoin.String(),
				appRelayMeter.maxCoin.String(),
			)
		}
	}

	appRelayMeter.numOverServicedRelays++
	numOverServicedRelays := appRelayMeter.numOverServicedRelays

	// Exponential backoff: log only when numOverServicedRelays is a power of 2
	shouldLog := (numOverServicedRelays & (numOverServicedRelays - 1)) == 0

	// Log the over-servicing warning.
	if shouldLog {
		rmtr.logger.Warn().Msgf(
			"overservicing enabled, application %q over-serviced %s",
			appRelayMeter.app.GetAddress(),
			overServicingCoin,
		)
	}

	return nil
}

// SetNonApplicableRelayReward updates the relay meter to make the relay reward for
// the given relay request as non-applicable.
// This is used when the relay is not volume / reward applicable but was optimistically
// accounted for in the relay meter.
func (rmtr *ProxyRelayMeter) SetNonApplicableRelayReward(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) error {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	sessionRelayMeter, ok := rmtr.sessionToRelayMeterMap[reqMeta.GetSessionHeader().GetSessionId()]
	if !ok {
		return ErrRelayerProxyUnknownSession.Wrap("session relay meter not found")
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCost, err := getSingleMinedRelayCostCoin(
		sessionRelayMeter.sharedParams,
		sessionRelayMeter.service,
		sessionRelayMeter.serviceRelayDifficulty,
	)
	if err != nil {
		return err
	}

	if sessionRelayMeter.numOverServicedRelays > 0 {
		return nil
	}

	// Decrease the consumed stake amount by relay cost.
	newConsumedAmount := sessionRelayMeter.consumedCoin.Sub(relayCost)

	sessionRelayMeter.consumedCoin = newConsumedAmount
	return nil
}

// forEachNewBlockFn is a callback function that is called every time a new block is committed.
// It resets the relay meter's application stakes every new session so that new
// application stakes can be metered.
func (rmtr *ProxyRelayMeter) forEachNewBlockFn(ctx context.Context, block client.Block) {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		return
	}

	// Delete the relay meters that correspond to settled sessions.
	for _, sessionRelayMeter := range rmtr.sessionToRelayMeterMap {
		sessionEndHeight := sessionRelayMeter.sessionHeader.GetSessionEndBlockHeight()
		sessionClaimOpenHeight := sessionEndHeight + int64(sharedParams.GetClaimWindowOpenOffsetBlocks())

		if block.Height() >= sessionClaimOpenHeight {
			// The session started its claim phase and the corresponding session relay meter
			// is no longer needed.
			delete(rmtr.sessionToRelayMeterMap, sessionRelayMeter.sessionHeader.GetSessionId())
		}
	}
}

// forEachEventApplicationStakedFn is a callback function that is called every time
// an application staked event is observed. It updates the relay meter known applications.
func (rmtr *ProxyRelayMeter) forEachEventApplicationStakedFn(ctx context.Context, event *apptypes.EventApplicationStaked) {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	app := event.GetApplication()

	// Since lean clients are supported, multiple suppliers might share the same RelayMiner.
	// Loop over all the suppliers that have metered the application and update their
	// max amount of stake they can consume.
	for _, sessionRelayMeter := range rmtr.sessionToRelayMeterMap {
		if sessionRelayMeter.app.Address != app.Address {
			continue
		}
		sessionRelayMeter.app.Stake = app.GetStake()
		appStakeShare := getAppStakePortionPayableToSessionSupplier(app.GetStake(), sessionRelayMeter.sharedParams)
		sessionRelayMeter.maxCoin = appStakeShare
	}
}

// ensureRequestSessionRelayMeter ensures that the relay miner has a relay meter
// ready for monitoring the requests's application's consumption.
func (rmtr *ProxyRelayMeter) ensureRequestSessionRelayMeter(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) (*sessionRelayMeter, error) {
	appAddress := reqMeta.GetSessionHeader().GetApplicationAddress()
	sessionId := reqMeta.GetSessionHeader().GetSessionId()

	relayMeter, ok := rmtr.sessionToRelayMeterMap[sessionId]
	// If the application is seen for the first time in this session, calculate the
	// max amount of stake the application can consume.
	if !ok {
		var app apptypes.Application
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

		service, err := rmtr.serviceQuerier.GetService(ctx, reqMeta.SessionHeader.ServiceId)
		if err != nil {
			return nil, err
		}

		serviceRelayDifficulty, err := rmtr.serviceQuerier.GetServiceRelayDifficulty(ctx, service.Id)
		if err != nil {
			return nil, err
		}

		// calculate the max amount of stake the application can consume in the current session.
		supplierAppStake := getAppStakePortionPayableToSessionSupplier(app.Stake, sharedParams)
		relayMeter = &sessionRelayMeter{
			app:                    app,
			consumedCoin:           cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
			maxCoin:                supplierAppStake,
			sessionHeader:          reqMeta.SessionHeader,
			sharedParams:           sharedParams,
			service:                &service,
			serviceRelayDifficulty: serviceRelayDifficulty,
		}

		rmtr.sessionToRelayMeterMap[sessionId] = relayMeter
	}

	return relayMeter, nil
}

// filterTypedEvents filters the provided events bytes for the typed event T.
// T is then filtered by the provided filter function.
func filterTypedEvents[T proto.Message](
	ctx context.Context,
	eventBzObs client.EventsBytesObservable,
	filterFn func(T) bool,
) observable.Observable[T] {
	eventObs, eventCh := channel.NewObservable[T]()
	channel.ForEach(ctx, eventBzObs, func(ctx context.Context, maybeTxBz either.Bytes) {
		if maybeTxBz.IsError() {
			return
		}
		txBz, _ := maybeTxBz.ValueOrError()

		// Try to deserialize the provided bytes into an abci.TxResult.
		txResult, err := tx.UnmarshalTxResult(txBz)
		if err != nil {
			return
		}

		for _, event := range txResult.Result.Events {
			eventApplicationStakedType := cosmostypes.MsgTypeURL(*new(T))
			if strings.Trim(event.GetType(), "/") != strings.Trim(eventApplicationStakedType, "/") {
				continue
			}

			typedEvent, err := cosmostypes.ParseTypedEvent(event)
			if err != nil {
				return
			}

			castedEvent, ok := typedEvent.(T)
			if !ok {
				return
			}

			// Apply the filter function to the typed event.
			if filterFn == nil || filterFn(castedEvent) {
				eventCh <- castedEvent
				return
			}
		}
	})

	return eventObs
}

// getSingleMinedRelayCostCoin returns the cost of a relay based on the shared parameters and the service.
// relayCost = Compute Units Per Relay (CUPR) * Compute Units To Token Multiplier (CUTTM) * relayDifficultyMultiplier
func getSingleMinedRelayCostCoin(
	sharedParams *sharedtypes.Params,
	service *sharedtypes.Service,
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) (cosmostypes.Coin, error) {
	// Get the difficulty multiplier based on the relay mining difficulty.
	difficultyTargetHash := relayMiningDifficulty.GetTargetHash()
	difficultyMultiplier := protocol.GetRelayDifficultyMultiplier(difficultyTargetHash)

	// Get the estimated cost of the relay if it gets mined.
	relayCostAmt := service.ComputeUnitsPerRelay * sharedParams.GetComputeUnitsToTokensMultiplier()
	relayCostRat := big.NewRat(int64(relayCostAmt), 1)
	estimatedRelayCostRat := big.NewRat(0, 1).Mul(relayCostRat, difficultyMultiplier)
	estimatedRelayCost := big.NewInt(0).Quo(estimatedRelayCostRat.Num(), estimatedRelayCostRat.Denom())

	estimatedRelayCostCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(estimatedRelayCost))

	return estimatedRelayCostCoin, nil
}

// getAppStakePortionPayableToSessionSupplier returns the portion of the application
// stake that can be consumed per supplier per session.
func getAppStakePortionPayableToSessionSupplier(
	stake *cosmostypes.Coin,
	sharedParams *sharedtypes.Params,
) cosmostypes.Coin {
	maxSuppliers := int64(sessionkeeper.NumSupplierPerSession)
	appStakePerSupplier := stake.Amount.Quo(math.NewInt(maxSuppliers))

	// Calculate the number of pending sessions that might consume the application's stake.
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	numBlocksUntilProofWindowCloses := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	pendingSessions := (numBlocksUntilProofWindowCloses + numBlocksPerSession - 1) / numBlocksPerSession

	appStakePerSessionSupplier := appStakePerSupplier.Quo(math.NewInt(pendingSessions))
	appStakePerSessionSupplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, appStakePerSessionSupplier)

	return appStakePerSessionSupplierCoin
}
