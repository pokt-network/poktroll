package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// EndBlockerUnbondApplications unbonds applications whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondApplications(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding applications at the end of the session.
	if currentHeight != k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight) {
		return nil
	}

	// Iterate over all applications and unbond the ones that have finished the unbonding period.
	// TODO_IMPROVE: Use an index to iterate over the applications that have initiated
	// the unbonding action instead of iterating over all of them.
	for _, application := range k.GetAllApplications(ctx) {
		// Ignore applications that have not initiated the unbonding action.
		if !application.IsUnbonding() {
			continue
		}

		unbondingHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &application)

		// If the unbonding height is ahead of the current height, the application
		// stays in the unbonding state.
		if unbondingHeight > currentHeight {
			continue
		}

		if err := k.UnbondApplication(ctx, &application); err != nil {
			return err
		}

		// TODO_NEXT(@bryanchriswhite): emit a new EventApplicationUnbondingEnd.
	}

	return nil
}

func (k Keeper) UnbondApplication(ctx context.Context, app *apptypes.Application) error {
	logger := k.Logger().With("method", "UnbondApplication")

	// Retrieve the account address of the application.
	appAddr, err := cosmostypes.AccAddressFromBech32(app.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", app.Address))
		return err
	}

	// Send the coins from the application pool back to the application.
	err = k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, apptypes.ModuleName, appAddr, []sdk.Coin{*app.Stake},
	)
	if err != nil {
		logger.Error(fmt.Sprintf(
			"could not send %v coins from module %s to account %s due to %v",
			app.Stake, appAddr, apptypes.ModuleName, err,
		))
		return err
	}

	// Remove the Application from the store.
	k.RemoveApplication(ctx, app.GetAddress())
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", app))

	return nil
}
