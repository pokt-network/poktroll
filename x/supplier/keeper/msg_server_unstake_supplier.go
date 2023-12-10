package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO(#73): Determine if an application needs an unbonding period after unstaking.
func (k msgServer) UnstakeSupplier(
	goCtx context.Context,
	msg *types.MsgUnstakeSupplier,
) (*types.MsgUnstakeSupplierResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "UnstakeSupplier")
	logger.Info(fmt.Sprintf("About to unstake supplier with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the supplier already exists or not
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.Address)
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Cannot unstake address %s", msg.Address))
		return nil, types.ErrSupplierNotFound
	}
	logger.Info(fmt.Sprintf("Supplier found. Unstaking supplier for address %s", msg.Address))

	// Retrieve the address of the supplier
	supplierAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, err
	}

	// Send the coins from the supplier pool back to the supplier
	err = k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, types.ModuleName, supplierAddress, []sdk.Coin{*supplier.Stake})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %s module to %s account due to %v", supplier.Stake, supplierAddress, types.ModuleName, err))
		return nil, err
	}

	// Update the Supplier in the store
	k.RemoveSupplier(ctx, supplierAddress.String())
	logger.Info(fmt.Sprintf("Successfully removed the supplier: %+v", supplier))
	return &types.MsgUnstakeSupplierResponse{}, nil
}
