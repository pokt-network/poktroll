package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) StakeApplication(ctx context.Context, msg *types.MsgStakeApplication) (*types.MsgStakeApplicationResponse, error) {
	logger := k.Logger().With("method", "StakeApplication")
	logger.Info(fmt.Sprintf("About to stake application with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("invalid MsgStakeApplication: %v", err))
		return nil, err
	}

	// Check if the application already exists or not
	var (
		err             error
		coinsToDelegate sdk.Coin
	)

	foundApp, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found. Creating new application for address %s", msg.Address))
		foundApp = k.createApplication(ctx, msg)
		coinsToDelegate = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Application found. Updating application for address %s", msg.Address))
		currAppStake := *foundApp.Stake
		if err = k.updateApplication(ctx, &foundApp, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = (*msg.Stake).Sub(currAppStake)
	}

	// Retrieve the address of the application
	appAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, err
	}

	// TODO_IMPROVE: Should we avoid making this call if `coinsToDelegate` = 0?
	// Send the coins from the application to the staked application pool
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, appAddress, types.ModuleName, []sdk.Coin{coinsToDelegate})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %s to %s module account due to %v", coinsToDelegate, appAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Application in the store
	k.SetApplication(ctx, foundApp)
	logger.Info(fmt.Sprintf("Successfully updated application stake for app: %+v", foundApp))

	return &types.MsgStakeApplicationResponse{}, nil
}

func (k msgServer) createApplication(
	_ context.Context,
	msg *types.MsgStakeApplication,
) types.Application {
	return types.Application{
		Address:                   msg.Address,
		Stake:                     msg.Stake,
		ServiceConfigs:            msg.Services,
		DelegateeGatewayAddresses: make([]string, 0),
	}
}

func (k msgServer) updateApplication(
	_ context.Context,
	app *types.Application,
	msg *types.MsgStakeApplication,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != app.Address {
		return types.ErrAppUnauthorized.Wrapf("msg Address (%s) != application address (%s)", msg.Address, app.Address)
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
