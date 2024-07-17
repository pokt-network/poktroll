package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

// TODO(#489): Determine if an application needs an unbonding period after unstaking.
func (k msgServer) UnstakeApplication(
	ctx context.Context,
	msg *application.MsgUnstakeApplication,
) (*application.MsgUnstakeApplicationResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"unstake_application",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UnstakeApplication")
	logger.Info(fmt.Sprintf("About to unstake application with msg: %v", msg))

	// Check if the application already exists or not
	var err error
	foundApp, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found. Cannot unstake address %s", msg.Address))
		return nil, application.ErrAppNotFound
	}
	logger.Info(fmt.Sprintf("Application found. Unstaking application for address %s", msg.Address))

	// Retrieve the address of the application
	appAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, err
	}

	// Send the coins from the application pool back to the application
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, appAddress, []sdk.Coin{*foundApp.Stake})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %s module to %s account due to %v", foundApp.Stake, appAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Application in the store
	k.RemoveApplication(ctx, appAddress.String())
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", foundApp))

	isSuccessful = true
	return &application.MsgUnstakeApplicationResponse{}, nil
}
