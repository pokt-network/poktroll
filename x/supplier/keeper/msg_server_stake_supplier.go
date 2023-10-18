package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier/types"
)

func (k msgServer) StakeSupplier(
	goCtx context.Context,
	msg *types.MsgStakeSupplier,
) (*types.MsgStakeSupplierResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "StakeSupplier")
	logger.Info("About to stake supplier with msg: %v", msg)

	// Check if the supplier already exists or not
	var err error
	var coinsToDelegate sdk.Coin
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.Address)
	if !isSupplierFound {
		logger.Info("Supplier not found. Creating new supplier for address %s", msg.Address)
		if err = k.createSupplier(ctx, &supplier, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = *msg.Stake
	} else {
		logger.Info("Supplier found. Updating supplier for address %s", msg.Address)
		currSupplierStake := *supplier.Stake
		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			return nil, err
		}
		coinsToDelegate = (*msg.Stake).Sub(currSupplierStake)
	}

	// Retrieve the address of the supplier
	supplierAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error("could not parse address %s", msg.Address)
		return nil, err
	}

	// Send the coins from the supplier to the staked supplier pool
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, supplierAddress, types.ModuleName, []sdk.Coin{coinsToDelegate})
	if err != nil {
		logger.Error("could not send %v coins from %s to %s module account due to %v", coinsToDelegate, supplierAddress, types.ModuleName, err)
		return nil, err
	}

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info("Successfully updated supplier stake for supplier: %+v", supplier)

	return &types.MsgStakeSupplierResponse{}, nil
}

func (k msgServer) createSupplier(
	ctx sdk.Context,
	supplier *sharedtypes.Supplier,
	msg *types.MsgStakeSupplier,
) error {
	*supplier = sharedtypes.Supplier{
		Address: msg.Address,
		Stake:   msg.Stake,
	}

	return nil
}

func (k msgServer) updateSupplier(
	ctx sdk.Context,
	supplier *sharedtypes.Supplier,
	msg *types.MsgStakeSupplier,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != supplier.Address {
		return sdkerrors.Wrapf(types.ErrSupplierUnauthorized, "msg Address (%s) != supplier address (%s)", msg.Address, supplier.Address)
	}

	if msg.Stake == nil {
		return sdkerrors.Wrapf(types.ErrSupplierInvalidStake, "stake amount cannot be nil")
	}

	if msg.Stake.IsLTE(*supplier.Stake) {

		return sdkerrors.Wrapf(types.ErrSupplierInvalidStake, "stake amount %v must be higher than previous stake amount %v", msg.Stake, supplier.Stake)
	}

	supplier.Stake = msg.Stake

	return nil
}
