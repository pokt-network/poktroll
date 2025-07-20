package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

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

	// Iterate over all unstaking applications and unbond the ones that have finished the unbonding period.
	// This iterator retrieves all applications that are in the unbonding state regardless of
	// whether their unbonding period has ended or not.
	// TODO_IMPROVE: Make this iterator more efficient by only retrieving applications
	// that have their unbonding period ended.
	allUnstakingApplicationsIterator := k.GetAllUnstakingApplicationsIterator(ctx)
	defer allUnstakingApplicationsIterator.Close()

	for ; allUnstakingApplicationsIterator.Valid(); allUnstakingApplicationsIterator.Next() {
		application, err := allUnstakingApplicationsIterator.Value()
		if err != nil {
			return err
		}

		// Ignore applications that have not initiated the unbonding action.
		if !application.IsUnbonding() {
			// If we are getting the application from the unbonding store and it is not
			// unbonding, this means that there is a dangling entry in the index.
			// log the error, remove the index entry but continue to the next supplier.
			logger.Error(fmt.Sprintf(
				"found application %s in unbonding store but it is not unbonding, removing index entry",
				application.Address,
			))
			k.removeApplicationUnstakingIndex(ctx, application.Address)
			continue
		}

		unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &application)

		// If the unbonding height is ahead of the current height, the application
		// stays in the unbonding state.
		if unbondingEndHeight > currentHeight {
			continue
		}

		if err := k.UnbondApplication(ctx, &application); err != nil {
			return err
		}

		sdkCtx = cosmostypes.UnwrapSDKContext(ctx)

		unbondingReason := apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_ELECTIVE
		if application.GetStake().Amount.LT(k.GetParams(ctx).MinStake.Amount) {
			unbondingReason = apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE
		}

		sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
		unbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
			ApplicationAddress: application.Address,
			Reason:             unbondingReason,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
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
		ctx, apptypes.ModuleName, appAddr, []cosmostypes.Coin{*app.Stake},
	)
	if err != nil {
		logger.Error(fmt.Sprintf(
			"could not send %v coins from module %s to account %s due to %v",
			app.Stake, appAddr, apptypes.ModuleName, err,
		))
		return err
	}

	// Remove the Application from the store.
	k.RemoveApplication(ctx, *app)
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", app))

	return nil
}
