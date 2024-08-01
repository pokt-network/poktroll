package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) StakeSupplier(ctx context.Context, msg *types.MsgStakeSupplier) (*types.MsgStakeSupplierResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"stake_supplier",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "StakeSupplier")
	logger.Info(fmt.Sprintf("About to stake supplier with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("invalid MsgStakeSupplier: %v", msg))
		return nil, err
	}

	// Check if the services the supplier is staking for exist
	for _, serviceConfig := range msg.Services {
		if _, serviceFound := k.serviceKeeper.GetService(ctx, serviceConfig.Service.Id); !serviceFound {
			logger.Error(fmt.Sprintf("service %q does not exist", serviceConfig.Service.Id))
			return nil, types.ErrSupplierServiceNotFound.Wrapf("service %q does not exist", serviceConfig.Service.Id)
		}
	}

	// Check if the supplier already exists or not
	var err error
	var coinsToEscrow sdk.Coin
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.Address)

	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %q", msg.Address))
		supplier = k.createSupplier(ctx, msg)

		// Ensure that only the owner can stake a new supplier.
		if err := supplier.EnsureOwner(msg.Sender); err != nil {
			logger.Error(fmt.Sprintf(
				"owner address %q in the message does not match the msg sender address %q",
				msg.OwnerAddress,
				msg.Sender,
			))

			return nil, err
		}

		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.Address))

		// Ensure that only the operator can update the supplier info.
		if err := supplier.EnsureOperator(msg.Sender); err != nil {
			logger.Error(fmt.Sprintf(
				"operator address %q in the message does not match the msg sender address %q",
				msg.Address,
				msg.Sender,
			))
			return nil, err
		}

		// Ensure that the owner addresses is not changed.
		if err := supplier.EnsureOwner(msg.OwnerAddress); err != nil {
			logger.Error("updating the supplier's owner address forbidden")

			return nil, err
		}

		// Ensure that the operator addresses is not changed.
		if err := supplier.EnsureOperator(msg.Address); err != nil {
			logger.Error("updating the supplier's operator address forbidden")

			return nil, err
		}

		currSupplierStake := *supplier.Stake
		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update supplier for address %q due to error %v", msg.Address, err))
			return nil, err
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currSupplierStake)
		if err != nil {
			return nil, err
		}
		logger.Info(fmt.Sprintf("Supplier is going to escrow an additional %+v coins", coinsToEscrow))

		// If the supplier has initiated an unstake action, cancel it since they are staking again.
		supplier.UnstakeSessionEndHeight = sharedtypes.SupplierNotUnstaking
	}

	// Must always stake or upstake (> 0 delta)
	if coinsToEscrow.IsZero() {
		logger.Warn(fmt.Sprintf("Supplier %q must escrow more than 0 additional coins", msg.Address))
		return nil, types.ErrSupplierInvalidStake.Wrapf("supplier %q must escrow more than 0 additional coins", msg.Address)
	}

	// Retrieve the account address of the supplier owner
	supplierOwnerAddress, err := sdk.AccAddressFromBech32(supplier.OwnerAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %q", supplier.OwnerAddress))
		return nil, err
	}

	// Send the coins from the supplier to the staked supplier pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, supplierOwnerAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %q to %q module account due to %v", coinsToEscrow, supplierOwnerAddress, types.ModuleName, err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, supplierOwnerAddress, types.ModuleName))

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	isSuccessful = true
	return &types.MsgStakeSupplierResponse{}, nil
}

// createSupplier creates a new supplier from the given message.
func (k msgServer) createSupplier(
	_ context.Context,
	msg *types.MsgStakeSupplier,
) sharedtypes.Supplier {
	return sharedtypes.Supplier{
		OwnerAddress: msg.OwnerAddress,
		Address:      msg.Address,
		Stake:        msg.Stake,
		Services:     msg.Services,
	}
}

// updateSupplier updates the given supplier with the given message.
func (k msgServer) updateSupplier(
	_ context.Context,
	supplier *sharedtypes.Supplier,
	msg *types.MsgStakeSupplier,
) error {
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
