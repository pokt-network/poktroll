package keeper

import (
	"context"
	"pocket/x/application/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StakeApplication(
	goCtx context.Context,
	msg *types.MsgStakeApplication,
) (*types.MsgStakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "StakeApplication")
	logger.Info("About to stake application with msg: %v", msg)

	// Check if the application already exists or not
	var err error
	var coinsToDelegate sdk.Coin
	app, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		logger.Info("Application not found. Creating new application for address %s", msg.Address)
		if err = k.createApplication(ctx, &app, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = *msg.Stake
	} else {
		logger.Info("Application found. Creating a new application for address %s", msg.Address)
		currAppStake := *app.Stake
		if err = k.updateApplication(ctx, &app, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = (*msg.Stake).Sub(currAppStake)
	}

	// Retrieve the address of the application
	appAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error("could not parse address %s", msg.Address)
		return nil, err
	}

	// Send the coins from the application to the staked application pool
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, appAddress, types.ModuleName, []sdk.Coin{coinsToDelegate})
	if err != nil {
		logger.Error("could not send coins %v coins from %s to %s module account due to %v", coinsToDelegate, appAddress, types.ModuleName, err)
		return nil, err
	}

	// Update the Application in the store
	k.SetApplication(ctx, app)
	logger.Info("Successfully updated application stake for app: %+v", app)

	return &types.MsgStakeApplicationResponse{}, nil
}

func (k msgServer) createApplication(
	ctx sdk.Context,
	app *types.Application,
	msg *types.MsgStakeApplication,
) error {
	*app = types.Application{
		Address: msg.Address,
		Stake:   msg.Stake,
	}

	return nil
}

func (k msgServer) updateApplication(
	ctx sdk.Context,
	app *types.Application,
	msg *types.MsgStakeApplication,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != app.Address {
		return errorsmod.Wrapf(types.ErrAppUnauthorized, "msg Address (%s) != application address (%s)", msg.Address, app.Address)
	}

	if msg.Stake == nil {
		return errorsmod.Wrapf(types.ErrAppInvalidStake, "stake amount cannot be nil")
	}

	if app.Stake.IsGTE(*msg.Stake) {
		return errorsmod.Wrapf(types.ErrAppStakeAmount, "stake amount %v must be higher than previous stake amount %v", msg.Stake, app.Stake)
	}

	app.Stake = msg.Stake

	return nil
}
