package keeper

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) StakeApplication(ctx context.Context, msg *types.MsgStakeApplication) (*types.MsgStakeApplicationResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"stake_application",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "StakeApplication")
	// Update the staking configurations of a existing app or stake a new app
	_, err := k.Keeper.StakeApplication(ctx, logger, msg)
	if err != nil {
		// DEV_NOTE: If the error is non-nil, StakeApplication SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	isSuccessful = true

	return &types.MsgStakeApplicationResponse{}, nil
}

// StakeApplication stakes (or updates) the application according to the given msg by applying the following logic:
//   - the msg is validated
//   - if the application is not found, it is created (in memory) according to the valid msg
//   - if the application is found and is not unbonding, it is updated (in memory) according to the msg
//   - if the application is found and is unbonding, it is updated (in memory; and no longer unbonding)
//   - additional stake validation (e.g. min stake, etc.)
//   - the positive difference between the msg stake and any current stake is transferred
//     from the staking application's account, to the application module's accounts.
//   - the (new or updated) application is persisted.
//   - an EventApplicationUnbondingCanceled event is emitted if the application was unbonding.
//   - an EventApplicationStaked event is emitted.
func (k Keeper) StakeApplication(
	ctx context.Context,
	logger cosmoslog.Logger,
	msg *types.MsgStakeApplication,
) (_ *types.Application, err error) {
	logger.Info(fmt.Sprintf("About to stake application with msg: %v", msg))

	if err = msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("invalid MsgStakeApplication: %v", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate per-session spend limit if a positive value is being set.
	if msg.PerSessionSpendLimit != nil && msg.PerSessionSpendLimit.IsPositive() {
		if msg.PerSessionSpendLimit.Amount.LT(types.MinPerSessionSpendLimit.Amount) {
			return nil, status.Error(
				codes.InvalidArgument,
				types.ErrAppInvalidStake.Wrapf(
					"per_session_spend_limit %s must be at least %s",
					msg.PerSessionSpendLimit, types.MinPerSessionSpendLimit,
				).Error(),
			)
		}
	}

	// Check if the application already exists or not
	var (
		coinsToEscrow   sdk.Coin
		wasAppUnbonding bool
	)
	foundApp, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found. Creating new application for address %q", msg.Address))
		foundApp = k.createApplication(ctx, msg)
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Application found. About to try and update application for address %q", msg.Address))
		currAppStake := *foundApp.Stake
		if err = k.updateApplication(ctx, &foundApp, msg); err != nil {
			logger.Info(fmt.Sprintf("could not update application for address %q due to error %v", msg.Address, err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currAppStake)
		if err != nil {
			logger.Info(fmt.Sprintf("could not calculate coins to escrow due to error %v", err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		logger.Info(fmt.Sprintf("Application is going to escrow an additional %+v coins", coinsToEscrow))

		// If the application has initiated an unstake action, cancel it since it is staking again.
		if foundApp.IsUnbonding() {
			wasAppUnbonding = true
			foundApp.UnstakeSessionEndHeight = types.ApplicationNotUnstaking
		}
	}

	// MUST ALWAYS stake or upstake (> 0 delta)
	if coinsToEscrow.IsZero() {
		logger.Warn(fmt.Sprintf("Application %q must escrow more than 0 additional coins", msg.Address))
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrAppInvalidStake.Wrapf(
				"application %q must escrow more than 0 additional coins",
				msg.Address,
			).Error())
	}

	// MUST ALWAYS have at least minimum stake.
	minStake := k.GetParams(ctx).MinStake
	// TODO_POST_MAINNET: If we support multiple native tokens, we will need to
	// start checking the denom here.
	if msg.Stake.Amount.LT(minStake.Amount) {
		err = fmt.Errorf("application %q must stake at least %s", msg.GetAddress(), minStake)
		logger.Info(err.Error())
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrAppInvalidStake.Wrapf("%s", err).Error(),
		)
	}

	// Retrieve the address of the application
	appAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Info(fmt.Sprintf("could not parse address %q", msg.Address))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Send the coins from the application to the staked application pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, appAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %q to %q module account due to %v", coinsToEscrow, appAddress, types.ModuleName, err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, appAddress, types.ModuleName))

	// Update the Application in the store
	k.SetApplication(ctx, foundApp)
	logger.Info(fmt.Sprintf("Successfully updated application stake for app: %+v", foundApp))

	// Collect events for emission.
	events := make([]sdk.Msg, 0)

	// If application unbonding was canceled, emit the corresponding event.
	if wasAppUnbonding {
		sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())
		events = append(events, &types.EventApplicationUnbondingCanceled{
			Application:      &foundApp,
			SessionEndHeight: sessionEndHeight,
		})
	}

	// ALWAYS emit an application staked event.
	currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	events = append(events, &types.EventApplicationStaked{
		Application:      &foundApp,
		SessionEndHeight: k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight),
	})

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err = sdkCtx.EventManager().EmitTypedEvents(events...); err != nil {
		err = types.ErrAppEmitEvent.Wrapf("(%+v): %s", events, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &foundApp, nil
}

func (k Keeper) createApplication(
	ctx context.Context,
	msg *types.MsgStakeApplication,
) types.Application {
	app := types.Application{
		Address:                   msg.Address,
		Stake:                     msg.Stake,
		ServiceConfigs:            msg.Services,
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations:      make(map[uint64]types.UndelegatingGatewayList),
		PerSessionSpendLimit:      normalizeSpendLimit(msg.PerSessionSpendLimit),
	}

	// A newly-staked application leaves service_config_history empty on purpose:
	// an empty history means "never changed", and GetActiveServiceConfigs falls
	// back to the flat ServiceConfigs for all heights. History is only written
	// once the app actually swaps its service (see recordApplicationServiceConfigChange).

	return app
}

// recordApplicationServiceConfigChange updates the application's service config
// to the provided services, recording the change in service_config_history only
// when the set of service IDs actually changes.
//
// Behavior:
//   - No service-membership change (stake bump, endpoint tweak): only the flat
//     ServiceConfigs snapshot is refreshed; history is left untouched. This keeps
//     "never changed" apps with empty history (GetActiveServiceConfigs falls back
//     to the flat snapshot).
//   - Service-membership change: the previously-active config is closed at the
//     next session start and the new config opens at the next session start, so
//     in-progress sessions stay deterministic (old active until nextSession, new
//     active from nextSession). On the FIRST ever change, the prior (flat) config
//     is first materialized into history as active since height 1, preserving the
//     pre-change timeline.
//
// History is never pruned (keep-forever) — see the proto comment on
// Application.service_config_history for why.
func (k Keeper) recordApplicationServiceConfigChange(
	ctx context.Context,
	app *types.Application,
	services []*sharedtypes.ApplicationServiceConfig,
) {
	// If the set of service IDs is unchanged, this is not a service swap: just
	// refresh the flat snapshot and leave history as-is.
	if appServiceIdSetEqual(app.ServiceConfigs, services) {
		app.ServiceConfigs = services
		return
	}

	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

	// First ever change: materialize the prior (flat) config into history as
	// active since height 1, so the pre-change period remains resolvable.
	if len(app.ServiceConfigHistory) == 0 {
		app.BackfillServiceConfigHistory()
	}

	updatedHistory := make([]*types.ApplicationServiceConfigUpdate, 0, len(app.ServiceConfigHistory)+len(services))

	// Carry forward existing history:
	// - Drop entries still scheduled to activate at this same next session boundary.
	//   Such an entry was opened earlier in the current session and has never served;
	//   this change supersedes it before it ever takes effect. It is re-opened below
	//   if its service is still in the new set. Dropping it (rather than deactivating
	//   it at its own activation height) avoids zero-width entries
	//   (activation == deactivation) that GenesisState.Validate rejects on re-import.
	// - Deactivate still-active entries at the next session start.
	for _, current := range app.ServiceConfigHistory {
		if current == nil || current.Service == nil {
			continue
		}

		if current.ActivationHeight == nextSessionStartHeight {
			continue
		}

		if current.DeactivationHeight == 0 {
			current.DeactivationHeight = nextSessionStartHeight
		}
		updatedHistory = append(updatedHistory, current)
	}

	// Open the new configs at the next session start.
	for _, svc := range services {
		updatedHistory = append(updatedHistory, &types.ApplicationServiceConfigUpdate{
			ApplicationAddress: app.Address,
			Service:            svc,
			ActivationHeight:   nextSessionStartHeight,
		})
	}

	app.ServiceConfigHistory = updatedHistory
	// Keep the flat snapshot in sync with the latest staked services.
	app.ServiceConfigs = services
}

// appServiceIdSetEqual reports whether two application service config slices
// cover the same set of service IDs.
func appServiceIdSetEqual(a, b []*sharedtypes.ApplicationServiceConfig) bool {
	if len(a) != len(b) {
		return false
	}
	ids := make(map[string]struct{}, len(a))
	for _, svc := range a {
		ids[svc.ServiceId] = struct{}{}
	}
	for _, svc := range b {
		if _, ok := ids[svc.ServiceId]; !ok {
			return false
		}
	}
	return true
}

// normalizeSpendLimit returns nil for nil or zero input (no limit),
// and the coin as-is for positive values.
func normalizeSpendLimit(limit *sdk.Coin) *sdk.Coin {
	if limit == nil || limit.IsZero() {
		return nil
	}
	if limit.Denom != "upokt" {
		return nil
	}
	return limit
}

func (k Keeper) updateApplication(
	ctx context.Context,
	app *types.Application,
	msg *types.MsgStakeApplication,
) error {
	// Checks if the msg address is the same as the current owner
	if msg.Address != app.Address {
		return types.ErrAppUnauthorized.Wrapf("msg Address %q != application address %q", msg.Address, app.Address)
	}

	// Validate that the stake is not being lowered
	if msg.Stake == nil {
		return types.ErrAppInvalidStake.Wrapf("stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*app.Stake) {
		return types.ErrAppInvalidStake.Wrapf("stake amount %v must be higher than previous stake amount %v", msg.Stake, app.Stake)
	}
	app.Stake = msg.Stake

	// Validate that the service configs maintain at least one service.
	// Additional validation is done in `msg.ValidateBasic` above.
	if len(msg.Services) == 0 {
		return types.ErrAppInvalidServiceConfigs.Wrapf("must have at least one service")
	}

	// Record the service-config change in history (only if the service set
	// actually changed; next-session activation) and refresh the flat
	// ServiceConfigs snapshot, instead of destructively overwriting it. This
	// preserves the previous config for historical session queries at past heights.
	k.recordApplicationServiceConfigChange(ctx, app, msg.Services)

	// Three-way per-session spend limit semantics:
	// nil = preserve existing limit, zero = clear limit, positive = set new limit.
	if msg.PerSessionSpendLimit != nil {
		if msg.PerSessionSpendLimit.IsZero() {
			app.PerSessionSpendLimit = nil // explicitly clear
		} else {
			app.PerSessionSpendLimit = msg.PerSessionSpendLimit
		}
	}

	return nil
}
