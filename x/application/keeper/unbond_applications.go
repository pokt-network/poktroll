package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
)

// EndBlockerUnbondApplications unbonds applications whose unbonding period has elapsed.
func (k Keeper) EndBlockerUnbondApplications(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	currentHeight := sdkCtx.BlockHeight()

	// Only process unbonding applications at the end of the session.
	if shared.IsSessionEndHeight(&sharedParams, currentHeight) {
		return nil
	}

	logger := k.Logger().With("method", "UnbondApplication")

	// Iterate over all applications and unbond the ones that have finished the unbonding period.
	// TODO_IMPROVE: Use an index to iterate over the applications that have initiated
	// the unbonding action instead of iterating over all of them.
	for _, application := range k.GetAllApplications(ctx) {
		// Ignore applications that have not initiated the unbonding action.
		if !application.IsUnbonding() {
			continue
		}

		unbondingHeight := types.GetApplicationUnbondingHeight(&sharedParams, &application)

		// If the unbonding height is ahead of the current height, the application
		// stays in the unbonding state.
		if unbondingHeight > currentHeight {
			continue
		}

		// Retrieve the account address of the application.
		applicationAccAddress, err := cosmostypes.AccAddressFromBech32(application.Address)
		if err != nil {
			logger.Error(fmt.Sprintf("could not parse address %s", application.Address))
			return err
		}

		// Send the coins from the application pool back to the application
		err = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, applicationAccAddress, []sdk.Coin{*application.Stake},
		)
		if err != nil {
			logger.Error(fmt.Sprintf(
				"could not send %v coins from module %s to account %s due to %v",
				application.Stake, applicationAccAddress, types.ModuleName, err,
			))
			return err
		}

		// Update the Application in the store
		k.RemoveApplication(ctx, applicationAccAddress.String())
		logger.Info(fmt.Sprintf("Successfully removed the application: %+v", application))

		sdkCtx = sdk.UnwrapSDKContext(ctx)
		unbondingBeginEvent := &types.EventApplicationUnbondingEnd{
			AppAddress: application.GetAddress(),
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(unbondingBeginEvent); err != nil {
			err = types.ErrAppEmitEvent.Wrapf("(%+v): %s", unbondingBeginEvent, err)
			logger.Error(err.Error())
		}
	}

	return nil
}
