package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// BeginBlockerActivateSupplierServices processes suppliers that have pending service activations
// at the current block height.
// It returns the number of suppliers whose services were activated.
func (k Keeper) BeginBlockerActivateSupplierServices(
	ctx context.Context,
) (numSuppliersWithServicesActivation uint64, err error) {
	logger := k.Logger().With("method", "ActivateSupplierServices")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Only activate supplier services at the start of a session.
	if !sharedtypes.IsSessionStartHeight(&sharedParams, currentBlockHeight) {
		return numSuppliersWithServicesActivation, nil
	}

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate services for suppliers",
		sharedtypes.GetSessionNumber(&sharedParams, currentBlockHeight),
	))

	events := make([]proto.Message, 0)

	// Iterate through all suppliers to check for pending service activations.
	// TODO_POST_MAINNET(@red-0ne): Optimize by using an index to track suppliers with pending activations.
	allSuppliersIterator := k.GetAllSuppliersIterator(ctx)
	defer allSuppliersIterator.Close()

	for ; allSuppliersIterator.Valid(); allSuppliersIterator.Next() {
		supplier, err := allSuppliersIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("could not get supplier from iterator: %v", err))
			return 0, err
		}

		// supplier.ServiceConfigHistory is guaranteed to contain at least one entry.
		// This is necessary for the session hydration process that relies on the
		// service config history to determine the current active service configuration.
		// It MUST be enforced by the methods that update the service config history.
		// (e.g. StakeSupplier, UnstakeSupplier, EndBlockerPruneSupplierServiceConfigHistory...)
		lastConfigIdx := len(supplier.ServiceConfigHistory) - 1
		// Check if this supplier has service config scheduled to activate at current height.
		latestConfig := supplier.ServiceConfigHistory[lastConfigIdx]
		if latestConfig.EffectiveBlockHeight == uint64(currentBlockHeight) {
			logger.Info(fmt.Sprintf(
				"activating services for supplier with operator address %q",
				supplier.OperatorAddress,
			))

			// Update supplier's active services to the new configuration
			supplier.Services = latestConfig.Services

			// Save the updated supplier.
			k.SetSupplier(ctx, supplier)
			numSuppliersWithServicesActivation += 1

			// Collect the event for the activated service configuration.
			event := &suppliertypes.EventSupplierServiceConfigActivated{
				Supplier:         &supplier,
				ActivationHeight: currentBlockHeight,
			}
			events = append(events, event)
		}
	}

	// Emit service activation events.
	if err := sdkCtx.EventManager().EmitTypedEvents(events...); err != nil {
		logger.Error(fmt.Sprintf("could not emit event %v", events))
		return 0, err
	}

	return numSuppliersWithServicesActivation, nil
}
