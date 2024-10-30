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
	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayMeter = (*ProxyRelayMeter)(nil)

// appRelayMeter is the relay meter's internal representation of an application's
// max and consumed stake.
type appRelayMeter struct {
	// The onchain application the relay meter is for.
	app apptypes.Application
	// The maximum uPOKT an application can pay this relayer for a given session.
	maxCoin cosmostypes.Coin
	// The amount of uPOKT a specific application has consumed from this relayer in the given session.
	consumedCoin cosmostypes.Coin
	// The current sessionHeader the application is metered in.
	sessionHeader *sessiontypes.SessionHeader

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
	// supplierToAppMetricsMap is a map of supplier addresses to application address
	// to the application's relay meter.
	// Only known applications (i.e. have already requested relaying) that have their stakes metered.
	// This map gets reset every new session in order to meter new applications, since the old
	// ones might have another Supplier set for their sessions.
	supplierToAppMetricsMap map[string]map[string]*appRelayMeter
	// overServicingAllowanceCoins allows Suppliers to overservice applications.
	// This entails providing a free service, to mine for relays, that they will not be paid for.
	// This is a common by some to build goodwill and receive a higher quality-of-service rating.
	// If negative, allow infinite overservicing.
	// TODO_MAINNET(@red-0ne): Expose overServicingAllowanceCoins as a configuration parameter.
	overServicingAllowanceCoins cosmostypes.Coin

	// relayMeterMu ensures that relay meter operations are thread-safe.
	relayMeterMu sync.Mutex

	applicationQuerier client.ApplicationQueryClient
	serviceQuerier     client.ServiceQueryClient
	sharedQuerier      client.SharedQueryClient
	eventsQueryClient  client.EventsQueryClient
	blockQuerier       client.BlockClient
}

func NewRelayMeter(deps depinject.Config) (relayer.RelayMeter, error) {
	overservicingAllowanceCoins := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000)
	rm := &ProxyRelayMeter{
		supplierToAppMetricsMap:     make(map[string]map[string]*appRelayMeter),
		overServicingAllowanceCoins: overservicingAllowanceCoins,
	}

	if err := depinject.Inject(
		deps,
		&rm.sharedQuerier,
		&rm.applicationQuerier,
		&rm.serviceQuerier,
		&rm.blockQuerier,
		&rm.eventsQueryClient,
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

	// Listen to application staked events and update known application stakes.
	// Since an applications might upstake (not downstake) during a session, this
	// stake increase is guaranteed to be available at settlement so it must be updated.
	// This also allows applications to adjust their stake mid-session and avoid
	// being rate limited or need to wait for the next session.
	// Stake updates take effect immediately.
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
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	// Create a metric if it does not exist.
	appMetrics, err := rmtr.ensureRequestAppMetrics(ctx, reqMeta, true)
	if err != nil {
		return err
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCostCoin, err := getMinedRelayCostCoin(
		appMetrics.sharedParams,
		appMetrics.service,
		appMetrics.serviceRelayDifficulty,
	)
	if err != nil {
		return err
	}

	// Increase the consumed stake amount by relay cost.
	newConsumedCoin := appMetrics.consumedCoin.Add(relayCostCoin)

	// If the consumed amount exceeds the max amount, return a rate limit error.
	overServicingLimited := !rmtr.overServicingAllowanceCoins.IsNegative()
	if overServicingLimited && appMetrics.maxCoin.IsLT(newConsumedCoin) {
		return ErrRelayerProxyRateLimited.Wrapf(
			"application has been rate limited, stake needed: %s, has: %s, ",
			newConsumedCoin.String(),
			appMetrics.maxCoin.String(),
		)
	}

	appMetrics.consumedCoin = newConsumedCoin
	return nil
}

// SetNonApplicableRelayReward updates the relay meter to make the relay reward for
// the given relay request as non-applicable.
// This is used when the relay is not volume / reward applicable but was optimistically
// accounted for in the relay meter.
func (rmtr *ProxyRelayMeter) SetNonApplicableRelayReward(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) error {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	appMetrics, err := rmtr.ensureRequestAppMetrics(ctx, reqMeta, false)
	if err != nil {
		return err
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCost, err := getMinedRelayCostCoin(
		appMetrics.sharedParams,
		appMetrics.service,
		appMetrics.serviceRelayDifficulty,
	)
	if err != nil {
		return err
	}

	// Decrease the consumed stake amount by relay cost.
	newConsumedAmount := appMetrics.consumedCoin.Sub(relayCost)

	appMetrics.consumedCoin = newConsumedAmount
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
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	// If the block observed is the last of the session, reset the relay meter's
	// to process next session's application requests.
	if block.Height()%numBlocksPerSession == 0 {
		rmtr.supplierToAppMetricsMap = make(map[string]map[string]*appRelayMeter)
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
	for supplierAddress := range rmtr.supplierToAppMetricsMap {
		appMetrics, ok := rmtr.supplierToAppMetricsMap[supplierAddress][app.GetAddress()]
		if !ok {
			continue
		}
		appMetrics.app.Stake = app.GetStake()
		appStakeShare := getApplicationStakeShare(app.GetStake(), appMetrics.sharedParams)
		appMetrics.maxCoin = appStakeShare.Add(rmtr.overServicingAllowanceCoins)
	}
}

func (rmtr *ProxyRelayMeter) ensureRequestAppMetrics(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata, createMetric bool) (*appRelayMeter, error) {
	appAddress := reqMeta.GetSessionHeader().GetApplicationAddress()
	supplierAddress := reqMeta.GetSupplierOperatorAddress()

	supplierApps, ok := rmtr.supplierToAppMetricsMap[supplierAddress]
	if !ok {
		rmtr.supplierToAppMetricsMap[supplierAddress] = make(map[string]*appRelayMeter)
		supplierApps = rmtr.supplierToAppMetricsMap[supplierAddress]
	}

	// Do not consider applications that have not been seen in this session.
	appMetrics, ok := supplierApps[appAddress]

	// If the application is seen for the first time in this session, calculate the
	// max amount of stake the application can consume.
	if !ok {
		var app apptypes.Application
		app, err := rmtr.applicationQuerier.GetApplication(ctx, appAddress)
		if err != nil {
			return nil, err
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

		if !createMetric {
			return nil, ErrRelayerProxyUnknownApplication.Wrap("required metric not found")
		}

		// calculate the max amount of stake the application can consume in the current session.
		appStakeShare := getApplicationStakeShare(app.Stake, sharedParams)
		maxAmount := appStakeShare.Add(rmtr.overServicingAllowanceCoins)
		appMetrics = &appRelayMeter{
			app:                    app,
			consumedCoin:           cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
			maxCoin:                maxAmount,
			sessionHeader:          reqMeta.SessionHeader,
			sharedParams:           sharedParams,
			service:                &service,
			serviceRelayDifficulty: serviceRelayDifficulty,
		}

		rmtr.supplierToAppMetricsMap[supplierAddress][appAddress] = appMetrics
	}

	return appMetrics, nil
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

// getMinedRelayCostCoin returns the cost of a relay based on the shared parameters and the service.
// relayCost = CUPR * CUTTM * relayDifficultyMultiplier
func getMinedRelayCostCoin(
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

	return cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(estimatedRelayCost)), nil
}

// getMinedRelayCost returns the share of the application's stake that can be consumed
// per supplier per session.
func getApplicationStakeShare(
	stake *cosmostypes.Coin,
	sharedParams *sharedtypes.Params,
) cosmostypes.Coin {
	maxRelayers := int64(sessionkeeper.NumSupplierPerSession)
	appStakePerSupplier := stake.Amount.Quo(math.NewInt(maxRelayers))

	// Calculate the number of pending sessions that might consume the application's stake.
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	pendingBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	pendingSessions := (pendingBlocks + numBlocksPerSession - 1) / numBlocksPerSession

	appStakePerSupplierSession := appStakePerSupplier.Quo(math.NewInt(pendingSessions))

	return cosmostypes.NewCoin(volatile.DenomuPOKT, appStakePerSupplierSession)
}
