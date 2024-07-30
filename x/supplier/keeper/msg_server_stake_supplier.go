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

	// Retain the previous operator address to check whether it changes after updating the supplier
	var previousOperatorAddr string
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %q", msg.Address))
		// Ensure that only supplier owner is able to stake.
		if err := ensureMsgSenderIsSupplierOwner(msg); err != nil {
			logger.Error(fmt.Sprintf(
				"owner address %q in the message does not match the msg sender address %q",
				msg.OwnerAddress,
				msg.Sender,
			))

			return nil, err
		}
		supplier = k.createSupplier(ctx, msg)
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.Address))

		// Only the owner can update the supplier's owner and operator addresses.
		if ownerOrOperatorAddressesChanged(msg, &supplier) {
			if err := ensureMsgSenderIsSupplierOwner(msg); err != nil {
				logger.Error("only the owner can update the supplier's owner and operator addresses")

				return nil, err
			}
		}

		currSupplierStake := *supplier.Stake
		// Get the previous operator address to update the store by deleting the old
		// key if it has been changed.
		previousOperatorAddr = supplier.Address
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
	supplierOwnerAddress, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %q", msg.OwnerAddress))
		return nil, err
	}

	// Send the coins from the supplier to the staked supplier pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, supplierOwnerAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %q to %q module account due to %v", coinsToEscrow, supplierOwnerAddress, types.ModuleName, err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, supplierOwnerAddress, types.ModuleName))

	// Remove the previous supplier key if the operator address has changed
	if hasDifferentOperatorAddr(previousOperatorAddr, &supplier) {
		k.RemoveSupplier(ctx, previousOperatorAddr)
	}
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
	// Check if the the msg owner address is the same as the current owner
	if msg.OwnerAddress != supplier.OwnerAddress {
		return types.ErrSupplierUnauthorized.Wrapf("msg OwnerAddress %q != supplier owner address %q", msg.OwnerAddress, supplier.OwnerAddress)
	}

	// Operator address should be already validated in `msg.ValidateBasic`.
	// TODO_CONSIDERATION: Delay the operator address change until the next session.
	supplier.Address = msg.Address

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

// hasDifferentOperatorAddr checks if the new operator address differs from the old one.
func hasDifferentOperatorAddr(oldOperatorAddress string, supplier *sharedtypes.Supplier) bool {
	if oldOperatorAddress == "" {
		return false
	}

	return oldOperatorAddress != supplier.Address
}

// ensureMsgSenderIsSupplierOwner returns an error if the message sender is not the supplier owner.
func ensureMsgSenderIsSupplierOwner(msg *types.MsgStakeSupplier) error {
	if msg.OwnerAddress == msg.Sender {
		types.ErrSupplierUnauthorized.Wrapf(
			"owner address %q in the message does not match the msg sender address %q",
			msg.OwnerAddress,
			msg.Sender,
		)
	}

	return nil
}

// ownerOrOperatorAddressesChanged checks if the owner or operator addresses have changed.
func ownerOrOperatorAddressesChanged(msg *types.MsgStakeSupplier, supplier *sharedtypes.Supplier) bool {
	return msg.OwnerAddress != supplier.OwnerAddress || msg.Address != supplier.Address
}
