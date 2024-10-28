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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayMeter = (*ProxyRelayMeter)(nil)

// appRelayMeter is the relay meter's internal representation of an application's
// max and consumed stake.
type appRelayMeter struct {
	maxAmount      cosmostypes.Coin
	consumedAmount cosmostypes.Coin
	app            apptypes.Application
}

// ProxyRelayMeter is the off-chain Supplier's rate limiter.
// It ensures that no application is over-serviced by the Supplier by maintaining
// the max amount of stake the supplier can consume per session and the amount of
// stake consumed by mined relays.
type ProxyRelayMeter struct {
	// The known applications that have their stakes metered.
	apps map[string]*appRelayMeter
	// overServicingAllowance adjusts the max amount to allow for controlled over-servicing
	// or, if negative, to create a stake buffer.
	// TODO_TECHDEBT(@red-0ne): Expose overServicingAllowance as a configuration parameter.
	overServicingAllowance int64

	applicationQuerier client.ApplicationQueryClient
	serviceQuerier     client.ServiceQueryClient
	sharedQuerier      client.SharedQueryClient
	eventsQueryClient  client.EventsQueryClient
	blockQuerier       client.BlockClient

	relayMeterMu sync.Mutex
}

func NewRelayMeter(deps depinject.Config) (relayer.RelayMeter, error) {
	rm := &ProxyRelayMeter{
		apps:                   make(map[string]*appRelayMeter),
		overServicingAllowance: 0,
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
	eventsObs, err := rmtr.eventsQueryClient.EventsBytes(ctx, "tm.event = 'Tx'")
	if err != nil {
		return err
	}

	// Listen to application staked events and update known application stakes.
	appStakedEvents := filterTypedEvents[*apptypes.EventApplicationStaked](ctx, eventsObs, nil)
	channel.ForEach(ctx, appStakedEvents, rmtr.forEachEventApplicationStakedFn)

	// Listen to new blocks and reset the relay meter application stakes every new session.
	committedBlocksSequence := rmtr.blockQuerier.CommittedBlocksSequence(ctx)
	channel.ForEach(ctx, committedBlocksSequence, rmtr.forEachNewBlockFn)

	return nil
}

// ClaimRelayPrice claims the relay price for the given relay request metadata.
// It deducts the relay cost from the application's stake and returns an error if
// the application has been rate limited.
func (rmtr *ProxyRelayMeter) ClaimRelayPrice(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) error {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		return err
	}

	service, err := rmtr.serviceQuerier.GetService(ctx, reqMeta.SessionHeader.ServiceId)
	if err != nil {
		return err
	}

	serviceRelayDifficulty, err := rmtr.serviceQuerier.GetServiceRelayDifficulty(ctx, service.Id)
	if err != nil {
		return err
	}

	appAddress := reqMeta.SessionHeader.ApplicationAddress

	appMetrics, ok := rmtr.apps[appAddress]
	// If the application is seen for the first time in this session, calculate the
	// max amount of stake the application can consume.
	if !ok {
		var app apptypes.Application
		app, err = rmtr.applicationQuerier.GetApplication(ctx, appAddress)
		if err != nil {
			return err
		}

		// calculate the max amount of stake the application can consume in the current session.
		appStakeShare := getApplicationStakeShare(app.Stake, sharedParams)
		maxAmount := appStakeShare.AddAmount(math.NewInt(rmtr.overServicingAllowance))
		appMetrics = &appRelayMeter{
			consumedAmount: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
			maxAmount:      maxAmount,
			app:            app,
		}
		rmtr.apps[appAddress] = appMetrics
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCost, err := getMinedRelayCost(sharedParams, &service, serviceRelayDifficulty)
	if err != nil {
		return err
	}

	// Increase the consumed stake amount by relay cost.
	newConsumedAmount := appMetrics.consumedAmount.Add(relayCost)

	// If the consumed amount exceeds the max amount, return a rate limit error.
	if appMetrics.maxAmount.IsLT(newConsumedAmount) {
		return ErrRelayerProxyRateLimited.Wrapf(
			"application has been rate limited, stake share: %s, expecting: %s",
			appMetrics.consumedAmount.String(),
			appMetrics.maxAmount.String(),
		)
	}

	appMetrics.consumedAmount = newConsumedAmount
	return nil
}

// UnclaimRelayPrice releases the claimed relay price back to the application's stake.
// This is because ClaimRelayPrice is optimistic and has to check against the application
// stake before serving the relay or check if it is reward / volume applicable.
// This method is called when the relay is not mined.
func (rmtr *ProxyRelayMeter) UnclaimRelayPrice(ctx context.Context, reqMeta servicetypes.RelayRequestMetadata) error {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	appAddress := reqMeta.SessionHeader.ApplicationAddress

	// Do not consider applications that have not been seen in this session.
	appMetrics, ok := rmtr.apps[appAddress]
	if !ok {
		return ErrRelayerProxyUnknownApplication
	}

	serviceId := reqMeta.SessionHeader.ServiceId

	sharedParams, err := rmtr.sharedQuerier.GetParams(ctx)
	if err != nil {
		return err
	}

	service, err := rmtr.serviceQuerier.GetService(ctx, serviceId)
	if err != nil {
		return err
	}

	difficulty, err := rmtr.serviceQuerier.GetServiceRelayDifficulty(ctx, serviceId)
	if err != nil {
		return err
	}

	// Get the cost of the relay based on the service and shared parameters.
	relayCost, err := getMinedRelayCost(sharedParams, &service, difficulty)
	if err != nil {
		return err
	}

	// Decrease the consumed stake amount by relay cost.
	newConsumedAmount := appMetrics.consumedAmount.Sub(relayCost)

	appMetrics.consumedAmount = newConsumedAmount
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

	if block.Height()%numBlocksPerSession == 0 {
		rmtr.apps = make(map[string]*appRelayMeter)
	}
}

// forEachEventApplicationStakedFn is a callback function that is called every time
// an application staked event is observed. It updates the relay meter's known
// application stakes.
func (rmtr *ProxyRelayMeter) forEachEventApplicationStakedFn(ctx context.Context, event *apptypes.EventApplicationStaked) {
	rmtr.relayMeterMu.Lock()
	defer rmtr.relayMeterMu.Unlock()

	app := event.GetApplication()
	if _, ok := rmtr.apps[app.GetAddress()]; !ok {
		return
	}

	rmtr.apps[app.GetAddress()].app.Stake = app.GetStake()
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

// getMinedRelayCost returns the cost of a relay based on the shared parameters and the service.
// relayCost = CUPR * CUTTM * relayDifficultyMultiplier
func getMinedRelayCost(
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
