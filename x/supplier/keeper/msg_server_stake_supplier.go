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
		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.Address))
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
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	isSuccessful = true
	return &types.MsgStakeSupplierResponse{}, nil
}

func (k msgServer) createSupplier(
	ctx context.Context,
	msg *types.MsgStakeSupplier,
) sharedtypes.Supplier {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight) + 1

	// Register activation height for each service. Since the supplier is new,
	// all services are activated at the end of the current session.
	servicesActivationHeight := make(map[string]uint64)
	for _, serviceConfig := range msg.Services {
		servicesActivationHeight[serviceConfig.Service.Id] = uint64(sessionEndHeight)
	}

	return sharedtypes.Supplier{
		Address:                  msg.Address,
		Stake:                    msg.Stake,
		Services:                 msg.Services,
		ServicesActivationHeight: servicesActivationHeight,
	}
}

func (k msgServer) updateSupplier(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	msg *types.MsgStakeSupplier,
) error {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != supplier.Address {
		return types.ErrSupplierUnauthorized.Wrapf("msg Address %q != supplier address %q", msg.Address, supplier.Address)
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

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight) + 1

	// Update activation height for services update. New services are activated at the
	// end of the current session, while existing ones keep their activation height.
	// TODO_CONSIDERAION: Service removal should take effect at the beginning of the
	// next session, otherwise sessions that are fetched at their start height may
	// still include Suppliers that no longer provide the services they removed.
	// For the same reason, any SupplierEndpoint change should take effect at the
	// beginning of the next session.
	servicesActivationHeight := make(map[string]uint64)
	for _, serviceConfig := range msg.Services {
		servicesActivationHeight[serviceConfig.Service.Id] = uint64(sessionEndHeight)
		// If the service has already been staked for, keep its activation height.
		for _, existingServiceConfig := range supplier.Services {
			if existingServiceConfig.Service.Id == serviceConfig.Service.Id {
				existingServiceActivationHeight := supplier.ServicesActivationHeight[serviceConfig.Service.Id]
				servicesActivationHeight[serviceConfig.Service.Id] = existingServiceActivationHeight
				break
			}
		}
	}

	supplier.Services = msg.Services
	supplier.ServicesActivationHeight = servicesActivationHeight

	return nil
}
