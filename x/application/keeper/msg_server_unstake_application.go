package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/types"
)

// TODO(#73): Determine if an application needs an unbonding period after unstaking.
func (k msgServer) UnstakeApplication(
	goCtx context.Context,
	msg *types.MsgUnstakeApplication,
) (*types.MsgUnstakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "UnstakeApplication")
	logger.Info("About to unstake application with msg: %v", msg)

	// Check if the application already exists or not
	var err error
	app, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		logger.Info("Application not found. Cannot unstake address %s", msg.Address)
		return nil, types.ErrAppNotFound
	}
	logger.Info("Application found. Unstaking application for address %s", msg.Address)

	// Retrieve the address of the application
	appAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error("could not parse address %s", msg.Address)
		return nil, err
	}

	// Send the coins from the application pool back to the application
	err = k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, appAddress, []sdk.Coin{*app.Stake})
	if err != nil {
		logger.Error("could not send %v coins from %s module to %s account due to %v", app.Stake, appAddress, types.ModuleName, err)
		return nil, err
	}

	// Update the Application in the store
	k.RemoveApplication(ctx, appAddress.String())
	logger.Info("Successfully removed the application: %+v", app)

	return &types.MsgUnstakeApplicationResponse{}, nil
}
