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
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_BETA(@red-0ne): Update supplier staking documentation to remove the upstaking requirement and introduce the staking fee.
func (k msgServer) StakeSupplier(
	ctx context.Context,
	msg *suppliertypes.MsgStakeSupplier,
) (*suppliertypes.MsgStakeSupplierResponse, error) {
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
				suppliertypes.ErrSupplierServiceNotFound.Wrapf("service %q does not exist", serviceConfig.ServiceId).Error(),
			)
		}
	}

	// Check if the supplier already exists or not
	var (
		err                  error
		wasSupplierUnbonding bool
		supplierCurrentStake sdk.Coin
	)
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.OperatorAddress)

	if !isSupplierFound {
		supplierCurrentStake = sdk.NewInt64Coin(volatile.DenomuPOKT, 0)
		logger.Info(fmt.Sprintf("Supplier not found. Creating new supplier for address %q", msg.OperatorAddress))
		supplier = k.createSupplier(ctx, msg)
	} else {
		logger.Info(fmt.Sprintf("Supplier found. About to try updating supplier with address %q", msg.OperatorAddress))

		supplierCurrentStake = *supplier.Stake

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

		if err = k.updateSupplier(ctx, &supplier, msg); err != nil {
			logger.Info(fmt.Sprintf("ERROR: could not update supplier for address %q due to error %v", msg.OperatorAddress, err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		// If the supplier has initiated an unstake action, cancel it since they are staking again.
		if supplier.UnstakeSessionEndHeight != sharedtypes.SupplierNotUnstaking {
			wasSupplierUnbonding = true
			supplier.UnstakeSessionEndHeight = sharedtypes.SupplierNotUnstaking
		}
	}

	// MUST ALWAYS have at least minimum stake.
	minStake := k.GetParams(ctx).MinStake
	if msg.Stake.Amount.LT(minStake.Amount) {
		err = suppliertypes.ErrSupplierInvalidStake.Wrapf(
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

	supplierStakingFee := k.GetParams(ctx).StakingFee

	if err = k.reconcileSupplierStakeDiff(ctx, msgSignerAddress, supplierCurrentStake, *msg.Stake); err != nil {
		logger.Error(fmt.Sprintf("Could not transfer supplier stake difference due to %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Deduct the staking fee from the supplier's account balance.
	// This is called after the stake difference is transferred to give the supplier
	// the opportunity to have enough balance to pay the fee.
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, msgSignerAddress, suppliertypes.ModuleName, sdk.NewCoins(*supplierStakingFee)); err != nil {
		logger.Info(fmt.Sprintf("ERROR: signer %q could not pay for the staking fee %s: %s", msgSignerAddress, supplierStakingFee, err))
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Update the Supplier in the store
	k.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("Successfully updated supplier stake for supplier: %+v", supplier))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := make([]sdk.Msg, 0)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	if wasSupplierUnbonding {
		events = append(events, &suppliertypes.EventSupplierUnbondingCanceled{
			Supplier:         &supplier,
			SessionEndHeight: sessionEndHeight,
			Height:           sdkCtx.BlockHeight(),
		})
	}

	// Emit an event which signals that the supplier staked.
	events = append(events, &suppliertypes.EventSupplierStaked{
		Supplier:         &supplier,
		SessionEndHeight: sessionEndHeight,
	})
	if err = sdkCtx.EventManager().EmitTypedEvents(events...); err != nil {
		err = suppliertypes.ErrSupplierEmitEvent.Wrapf("(%+v): %s", events, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &suppliertypes.MsgStakeSupplierResponse{
		Supplier: &supplier,
	}, nil
}

// createSupplier creates a new supplier from the given message.
func (k msgServer) createSupplier(
	ctx context.Context,
	msg *suppliertypes.MsgStakeSupplier,
) sharedtypes.Supplier {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

	supplier := sharedtypes.Supplier{
		OwnerAddress:    msg.OwnerAddress,
		OperatorAddress: msg.OperatorAddress,
		Stake:           msg.Stake,
		// The supplier won't be active until the start of the next session.
		// This is to ensure that it doesn't pop up in session hydrations and potentially
		// evicting another supplier.
		Services:             make([]*sharedtypes.SupplierServiceConfig, 0),
		ServiceConfigHistory: make([]*sharedtypes.ServiceConfigUpdate, 0),
	}

	// Store the service configurations details of the newly created supplier.
	// They will take effect at the start of the next session.
	servicesUpdate := &sharedtypes.ServiceConfigUpdate{
		Services: msg.Services,
		// The effective block height is the start of the next session.
		EffectiveBlockHeight: uint64(nextSessionStartHeight),
	}
	supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, servicesUpdate)

	return supplier
}

// updateSupplier updates the given supplier with the given message.
func (k msgServer) updateSupplier(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	msg *suppliertypes.MsgStakeSupplier,
) error {
	// Validate that the stake is not being lowered
	if msg.Stake == nil {
		return suppliertypes.ErrSupplierInvalidStake.Wrapf("stake amount cannot be nil")
	}

	supplier.Stake = msg.Stake
	supplier.OwnerAddress = msg.OwnerAddress

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

	// Store the details of the supplier's new service configurations.
	servicesUpdate := &sharedtypes.ServiceConfigUpdate{
		Services: msg.Services,
		// The effective block height is the start of the next session.
		EffectiveBlockHeight: uint64(nextSessionStartHeight),
	}

	configUpdateLog := supplier.ServiceConfigHistory
	latestUpdateIdx := len(configUpdateLog) - 1
	// Overwrite the latest service configuration if there is already a service
	// config update for the same session start height.
	if latestUpdateIdx >= 0 && configUpdateLog[latestUpdateIdx].EffectiveBlockHeight == uint64(nextSessionStartHeight) {
		supplier.ServiceConfigHistory[latestUpdateIdx] = servicesUpdate
		return nil
	}

	// Otherwise, append the new service configuration update.
	supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, servicesUpdate)
	return nil
}

// reconcileSupplierStakeDiff transfers the difference between the current and new stake
// amounts by either escrowing, when the stake is increased, or unescrowing otherwise.
func (k msgServer) reconcileSupplierStakeDiff(
	ctx context.Context,
	signerAddr sdk.AccAddress,
	currentStake sdk.Coin,
	newStake sdk.Coin,
) error {
	logger := k.Logger().With("method", "reconcileSupplierStakeDiff")

	// The Supplier is increasing its stake, so escrow the difference
	if currentStake.Amount.LT(newStake.Amount) {
		coinsToEscrow := sdk.NewCoins(newStake.Sub(currentStake))

		// Send the coins from the message signer account to the staked supplier pool
		return k.bankKeeper.SendCoinsFromAccountToModule(ctx, signerAddr, suppliertypes.ModuleName, coinsToEscrow)
	}

	// Ensure that the new stake is at least the minimum stake which is required for:
	// 1. The supplier to be considered active.
	// 2. Cover for any potential slashing that may occur during claims settlement.
	minStake := k.GetParams(ctx).MinStake
	if newStake.Amount.LT(minStake.Amount) {
		err := suppliertypes.ErrSupplierInvalidStake.Wrapf(
			"supplier with owner %q must stake at least %s",
			signerAddr, minStake,
		)
		return err
	}

	// The supplier is decreasing its stake, unescrow the difference.
	if currentStake.Amount.GT(newStake.Amount) {
		coinsToUnescrow := sdk.NewCoins(currentStake.Sub(newStake))

		// Send the coins from the staked supplier pool to the message signer account
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, signerAddr, coinsToUnescrow)
	}

	// The supplier is not changing its stake. This can happen if the supplier
	// is updating its service configurations or owner address but not the stake.
	logger.Info(fmt.Sprintf("Updating supplier with address %q but stake is unchanged", signerAddr.String()))
	return nil
}
