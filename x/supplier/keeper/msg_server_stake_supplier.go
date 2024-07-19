package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) StakeSupplier(ctx context.Context, msg *supplier.MsgStakeSupplier) (*supplier.MsgStakeSupplierResponse, error) {
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
			return nil, supplier.ErrSupplierServiceNotFound.Wrapf("service %q does not exist", serviceConfig.Service.Id)
		}
	}

	// Check if the supplier already exists or not
	var err error
	var coinsToEscrow sdk.Coin
	supplierToStake, isSupplierFound := k.GetSupplier(ctx, msg.Address)

	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %q", msg.Address))
		supplierToStake = k.createSupplier(ctx, msg)
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.Address))
		currSupplierStake := *supplierToStake.Stake
		if err = k.updateSupplier(ctx, &supplierToStake, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update supplier for address %q due to error %v", msg.Address, err))
			return nil, err
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currSupplierStake)
		if err != nil {
			return nil, err
		}
		logger.Info(fmt.Sprintf("Supplier is going to escrow an additional %+v coins", coinsToEscrow))
	}

	// Must always stake or upstake (> 0 delta)
	if coinsToEscrow.IsZero() {
		logger.Warn(fmt.Sprintf("Supplier %q must escrow more than 0 additional coins", msg.Address))
		return nil, supplier.ErrSupplierInvalidStake.Wrapf("supplier %q must escrow more than 0 additional coins", msg.Address)
	}

	// Retrieve the address of the supplier
	supplierAddress, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %q", msg.Address))
		return nil, err
	}

	// Send the coins from the supplier to the staked supplier pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, supplierAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %q to %q module account due to %v", coinsToEscrow, supplierAddress, types.ModuleName, err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, supplierAddress, types.ModuleName))

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplierToStake)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplierToStake))

	isSuccessful = true
	return &supplier.MsgStakeSupplierResponse{}, nil
}

func (k msgServer) createSupplier(
	_ context.Context,
	msg *supplier.MsgStakeSupplier,
) shared.Supplier {
	return shared.Supplier{
		Address:  msg.Address,
		Stake:    msg.Stake,
		Services: msg.Services,
	}
}

func (k msgServer) updateSupplier(
	_ context.Context,
	supplierToUpdate *shared.Supplier,
	msg *supplier.MsgStakeSupplier,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != supplierToUpdate.Address {
		return supplier.ErrSupplierUnauthorized.Wrapf("msg Address %q != supplier address %q", msg.Address, supplierToUpdate.Address)
	}

	// Validate that the stake is not being lowered
	if msg.Stake == nil {
		return supplier.ErrSupplierInvalidStake.Wrapf("stake amount cannot be nil")
	}
	if msg.Stake.IsLTE(*supplierToUpdate.Stake) {

		return supplier.ErrSupplierInvalidStake.Wrapf("stake amount %v must be higher than previous stake amount %v", msg.Stake, supplierToUpdate.Stake)
	}
	supplierToUpdate.Stake = msg.Stake

	// Validate that the service configs maintain at least one service.
	// Additional validation is done in `msg.ValidateBasic` above.
	if len(msg.Services) == 0 {
		return supplier.ErrSupplierInvalidServiceConfig.Wrapf("must have at least one service")
	}
	supplierToUpdate.Services = msg.Services

	return nil
}
