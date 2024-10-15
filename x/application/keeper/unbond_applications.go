package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// EndBlockerUnbondApplications unbonds applications whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondApplications(ctx context.Context) error {
	logger := k.Logger().With("method", "EndBlockerUnbondApplications")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding applications at the end of the session.
	if sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
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

		sdkCtx = sdk.UnwrapSDKContext(ctx)

		unbondingReason := apptypes.ApplicationUnbondingReason_ELECTIVE
		if application.GetUnstakeSessionEndHeight() == apptypes.ApplicationBelowMinStake {
			unbondingReason = apptypes.ApplicationUnbondingReason_BELOW_MIN_STAKE
		}

		unbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
			Application: &application,
			Reason:      unbondingReason,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(unbondingEndEvent); err != nil {
			err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", unbondingEndEvent, err)
			logger.Error(err.Error())
			return err
		}
	}

	return nil
}

// UnbondApplication transfers the application stake to the bank module balance for the
// corresponding account and removes the application from the application module state.
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
