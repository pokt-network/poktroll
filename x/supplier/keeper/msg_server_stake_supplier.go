package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_IMPROVE(@Olshansk): Add more logging to staking & unstaking branches (success, failure, etc...).

func (k msgServer) StakeSupplier(ctx context.Context, msg *types.MsgStakeSupplier) (*types.MsgStakeSupplierResponse, error) {
	logger := k.Logger().With("method", "StakeSupplier")
	logger.Info(fmt.Sprintf("About to stake supplier with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("invalid MsgStakeSupplier: %v", msg))
		return nil, err
	}

	// Check if the supplier already exists or not
	var err error
	var coinsToDelegate sdk.Coin
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.Address)

	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %s", msg.Address))
		supplier = k.createSupplier(ctx, msg)
		coinsToDelegate = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try and update supplier with address %s", msg.Address))
		currSupplierStake := *supplier.Stake
		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update supplier for address %s due to error %v", msg.Address, err))
			return nil, err
		}
		coinsToDelegate, err = (*msg.Stake).SafeSub(currSupplierStake)
		logger.Debug(fmt.Sprintf("Supplier is going to delegate an additional %+v coins", coinsToDelegate))
		if err != nil {
			return nil, err
		}
	}

	// Must always stake or upstake (> 0 delta)
	if coinsToDelegate.IsZero() {
		logger.Warn(fmt.Sprintf("Supplier %s must delegate more than 0 additional coins", msg.Address))
		return nil, types.ErrSupplierInvalidStake.Wrapf("supplier %s must delegate more than 0 additional coins", msg.Address)
	}

	// Retrieve the address of the supplier
	supplierAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, err
	}

	// TODO_IMPROVE: Should we avoid making this call if `coinsToDelegate` = 0?
	// Send the coins from the supplier to the staked supplier pool
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, supplierAddress, types.ModuleName, []sdk.Coin{coinsToDelegate})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %s to %s module account due to %v", coinsToDelegate, supplierAddress, types.ModuleName, err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Successfully sent %v coins from %s to %s module account", coinsToDelegate, supplierAddress, types.ModuleName))

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	return &types.MsgStakeSupplierResponse{}, nil
}

func (k msgServer) createSupplier(
	_ context.Context,
	msg *types.MsgStakeSupplier,
) sharedtypes.Supplier {
	return sharedtypes.Supplier{
		Address:  msg.Address,
		Stake:    msg.Stake,
		Services: msg.Services,
	}
}

func (k msgServer) updateSupplier(
	_ context.Context,
	supplier *sharedtypes.Supplier,
	msg *types.MsgStakeSupplier,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != supplier.Address {
		return types.ErrSupplierUnauthorized.Wrapf("msg Address (%s) != supplier address (%s)", msg.Address, supplier.Address)
	}

	// Validate that the stake is not being lowered
	if msg.Stake == nil {
		return types.ErrSupplierInvalidStake.Wrapf("stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*supplier.Stake) {

		return types.ErrSupplierInvalidStake.Wrapf("stake amount %v must be higher than previous stake amount %v", msg.Stake, supplier.Stake)
	}
	supplier.Stake = msg.Stake

	// Validate that the service configs maintain at least one service.
	// Additional validation is done in `msg.ValidateBasic` above.
	if len(msg.Services) == 0 {
		return types.ErrSupplierInvalidServiceConfig.Wrapf("must have at least one service")
	}
	supplier.Services = msg.Services

	return nil
}
