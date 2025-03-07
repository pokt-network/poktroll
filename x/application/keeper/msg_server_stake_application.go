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
	stakedApp, err := k.Keeper.StakeApplication(ctx, logger, msg)
	if err != nil {
		// DEV_NOTE: If the error is non-nil, StakeApplication SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	isSuccessful = true

	return &types.MsgStakeApplicationResponse{
		Application: stakedApp,
	}, nil
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
	_ context.Context,
	msg *types.MsgStakeApplication,
) types.Application {
	return types.Application{
		Address:                   msg.Address,
		Stake:                     msg.Stake,
		ServiceConfigs:            msg.Services,
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations:      make(map[uint64]types.UndelegatingGatewayList),
	}
}

func (k Keeper) updateApplication(
	_ context.Context,
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
	app.ServiceConfigs = msg.Services

	return nil
}
