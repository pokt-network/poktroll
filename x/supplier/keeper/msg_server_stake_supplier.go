package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/telemetry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

var (
	// TODO_BETA: Make supplier staking fee a governance parameter
	// TODO_BETA(@red-0ne): Update supplier staking documentation to remove the upstaking
	// requirement and introduce the staking fee.
	SupplierStakingFee = sdk.NewInt64Coin(volatile.DenomuPOKT, 1)
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
		logger.Info(fmt.Sprintf("invalid MsgStakeSupplier: %v", msg))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the services the supplier is staking for exist
	for _, serviceConfig := range msg.Services {
		if _, serviceFound := k.serviceKeeper.GetService(ctx, serviceConfig.ServiceId); !serviceFound {
			logger.Info(fmt.Sprintf("service %q does not exist", serviceConfig.ServiceId))
			return nil, status.Error(
				codes.InvalidArgument,
				types.ErrSupplierServiceNotFound.Wrapf("service %q does not exist", serviceConfig.ServiceId).Error(),
			)
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
			return nil, status.Error(
				codes.InvalidArgument,
				sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
					"signer address %s does not match owner address %s or supplier operator address %s",
					msg.Signer, msg.OwnerAddress, msg.OperatorAddress,
				).Error(),
			)
		}

		// Ensure that only the owner can change the OwnerAddress.
		// (i.e. fail if owner address changed and the owner is not the msg signer)
		if !supplier.HasOwner(msg.OwnerAddress) && !msg.IsSigner(supplier.OwnerAddress) {
			err = sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
				"signer %q is not allowed to update the owner address %q",
				msg.Signer, supplier.OwnerAddress,
			)
			logger.Info(fmt.Sprintf("ERROR: %s", err))

			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		// Ensure that the operator addresses cannot be changed. This is because changing
		// it mid-session invalidates the current session.
		if !supplier.HasOperator(msg.OperatorAddress) {
			err = sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrap(
				"updating the operator address is forbidden, unstake then re-stake with the updated operator address",
			)
			logger.Info(fmt.Sprintf("ERROR: %s", err))

			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		currSupplierStake := *supplier.Stake
		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			logger.Info(fmt.Sprintf("ERROR: could not update supplier for address %q due to error %v", msg.OperatorAddress, err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		coinsToEscrow, err = (*msg.Stake).SafeSub(currSupplierStake)
		if err != nil {
			logger.Info(fmt.Sprintf("ERROR: %s", err))
			return nil, status.Error(codes.Internal, err.Error())
		}
		logger.Info(fmt.Sprintf("Supplier is going to escrow an additional %+v coins", coinsToEscrow))

		// If the supplier has initiated an unstake action, cancel it since they are staking again.
		supplier.UnstakeSessionEndHeight = sharedtypes.SupplierNotUnstaking
	}

	// TODO_BETA: Remove requirement of MUST ALWAYS stake or upstake (>= 0 delta)
	// TODO_POST_MAINNET: Should we allow stake decrease down to min stake?
	if coinsToEscrow.IsNegative() {
		err = types.ErrSupplierInvalidStake.Wrapf(
			"Supplier signer %q stake (%s) must be greater than or equal to the current stake (%s)",
			msg.Signer, msg.GetStake(), supplier.Stake,
		)
		logger.Info(fmt.Sprintf("WARN: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// MUST ALWAYS have at least minimum stake.
	minStake := k.GetParams(ctx).MinStake
	if msg.Stake.Amount.LT(minStake.Amount) {
		err = types.ErrSupplierInvalidStake.Wrapf(
			"supplier with owner %q must stake at least %s",
			msg.GetOwnerAddress(), minStake,
		)
		logger.Info(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Retrieve the account address of the message signer
	msgSignerAddress, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		logger.Info(fmt.Sprintf("ERROR: could not parse address %q", msg.Signer))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Send the coins from the message signer account to the staked supplier pool
	stakeWithFee := sdk.NewCoins(coinsToEscrow.Add(SupplierStakingFee))
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, msgSignerAddress, types.ModuleName, stakeWithFee)
	if err != nil {
		logger.Info(fmt.Sprintf("ERROR: could not send %v coins from %q to %q module account due to %v", coinsToEscrow, msgSignerAddress, types.ModuleName, err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	logger.Info(fmt.Sprintf("Successfully escrowed %v coins from %q to %q module account", coinsToEscrow, msgSignerAddress, types.ModuleName))

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Emit an event which signals that the supplier staked.
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
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

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

	// TODO_BETA: No longer require upstaking. Remove this check.
	if msg.Stake.IsLT(*supplier.Stake) {
		return types.ErrSupplierInvalidStake.Wrapf(
			"stake amount %v must be greater than or equal than previous stake amount %v",
			msg.Stake, supplier.Stake,
		)
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
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

	// Update activation height for services update. New services are activated at the
	// end of the current session, while existing ones keep their activation height.
	// TODO_MAINNET: Service removal should take effect at the beginning of the
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
