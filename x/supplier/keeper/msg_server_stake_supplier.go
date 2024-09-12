package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/shared"
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

	// ValidateBasic also validates that the msg signer is the owner or operator of the supplier
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("invalid MsgStakeSupplier: %v", msg))
		return nil, err
	}

	// Check if the services the supplier is staking for exist
	for _, serviceConfig := range msg.Services {
		if _, serviceFound := k.serviceKeeper.GetService(ctx, serviceConfig.ServiceId); !serviceFound {
			logger.Error(fmt.Sprintf("service %q does not exist", serviceConfig.ServiceId))
			return nil, types.ErrSupplierServiceNotFound.Wrapf("service %q does not exist", serviceConfig.ServiceId)
		}
	}

	// Check if the supplier already exists or not
	var err error
	var coinsToEscrow sdk.Coin
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.OperatorAddress)

	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %q", msg.OperatorAddress))
		supplier = k.createSupplier(ctx, msg)

		coinsToEscrow = *msg.Stake
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.OperatorAddress))

		// Ensure the signer is either the owner or the operator of the supplier.
		if !msg.IsSigner(supplier.OwnerAddress) && !msg.IsSigner(supplier.OperatorAddress) {
			return nil, sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
				"signer address %s does not match owner address %s or supplier operator address %s",
				msg.Signer,
				msg.OwnerAddress,
				msg.OperatorAddress,
			)
		}

		// Ensure that only the owner can change the OwnerAddress.
		// (i.e. fail if owner address changed and the owner is not the msg signer)
		if !supplier.HasOwner(msg.OwnerAddress) && !msg.IsSigner(supplier.OwnerAddress) {
			logger.Error("only the supplier owner is allowed to update the owner address")

			return nil, sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
				"signer %q is not allowed to update the owner address %q",
				msg.Signer,
				supplier.OwnerAddress,
			)
		}

		// Ensure that the operator addresses cannot be changed. This is because changing
		// it mid-session invalidates the current session.
		if !supplier.HasOperator(msg.OperatorAddress) {
			logger.Error("updating the supplier's operator address forbidden")

			return nil, sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrap(
				"updating the operator address is forbidden, unstake then re-stake with the updated operator address",
			)
		}

		currSupplierStake := *supplier.Stake
		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			logger.Error(fmt.Sprintf("could not update supplier for address %q due to error %v", msg.OperatorAddress, err))
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
		logger.Warn(fmt.Sprintf("Signer %q must escrow more than 0 additional coins", msg.Signer))
		return nil, types.ErrSupplierInvalidStake.Wrapf("Signer %q must escrow more than 0 additional coins", msg.Signer)
	}

	// Retrieve the account address of the message signer
	msgSignerAddress, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %q", msg.Signer))
		return nil, err
	}

	// Send the coins from the message signer account to the staked supplier pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, msgSignerAddress, types.ModuleName, []sdk.Coin{coinsToEscrow})
	if err != nil {
		logger.Error(fmt.Sprintf("could not send %v coins from %q to %q module account due to %v", coinsToEscrow, msgSignerAddress, types.ModuleName, err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, msgSignerAddress, types.ModuleName))

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	event := &types.EventSupplierStaked{
		Supplier: &supplier,
	}
	if eventErr := sdkCtx.EventManager().EmitTypedEvent(event); eventErr != nil {
		logger.Error(fmt.Sprintf("failed to emit event: %+v; %s", event, eventErr))
	}

	isSuccessful = true
	return &types.MsgStakeSupplierResponse{}, nil
}

// createSupplier creates a new supplier from the given message.
func (k msgServer) createSupplier(
	ctx context.Context,
	msg *types.MsgStakeSupplier,
) sharedtypes.Supplier {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	nextSessionStartHeight := shared.GetNextSessionStartHeight(&sharedParams, currentHeight)

	// Register activation height for each service. Since the supplier is new,
	// all services are activated at the end of the current session.
	servicesActivationHeightsMap := make(map[string]uint64)
	for _, serviceConfig := range msg.Services {
		servicesActivationHeightsMap[serviceConfig.ServiceId] = uint64(nextSessionStartHeight)
	}

	return sharedtypes.Supplier{
		OwnerAddress:                 msg.OwnerAddress,
		OperatorAddress:              msg.OperatorAddress,
		Stake:                        msg.Stake,
		Services:                     msg.Services,
		ServicesActivationHeightsMap: servicesActivationHeightsMap,
	}
}

// updateSupplier updates the given supplier with the given message.
func (k msgServer) updateSupplier(
	ctx context.Context,
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

	supplier.OwnerAddress = msg.OwnerAddress

	// Validate that the service configs maintain at least one service.
	// Additional validation is done in `msg.ValidateBasic` above.
	if len(msg.Services) == 0 {
		return types.ErrSupplierInvalidServiceConfig.Wrapf("must have at least one service")
	}

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	nextSessionStartHeight := shared.GetNextSessionStartHeight(&sharedParams, currentHeight)

	// Update activation height for services update. New services are activated at the
	// end of the current session, while existing ones keep their activation height.
	// TODO_CONSIDERAION: Service removal should take effect at the beginning of the
	// next session, otherwise sessions that are fetched at their start height may
	// still include Suppliers that no longer provide the services they removed.
	// For the same reason, any SupplierEndpoint change should take effect at the
	// beginning of the next session.
	ServicesActivationHeightMap := make(map[string]uint64)
	for _, serviceConfig := range msg.Services {
		ServicesActivationHeightMap[serviceConfig.ServiceId] = uint64(nextSessionStartHeight)
		// If the service has already been staked for, keep its activation height.
		for _, existingServiceConfig := range supplier.Services {
			if existingServiceConfig.ServiceId == serviceConfig.ServiceId {
				existingServiceActivationHeight := supplier.ServicesActivationHeightsMap[serviceConfig.ServiceId]
				ServicesActivationHeightMap[serviceConfig.ServiceId] = existingServiceActivationHeight
				break
			}
		}
	}

	supplier.Services = msg.Services
	supplier.ServicesActivationHeightsMap = ServicesActivationHeightMap

	return nil
}
