package keeper

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/telemetry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// StakeSupplier processes a MsgStakeSupplier message from a supplier who wants to stake tokens
// and offer services on the network. This function handles both initial staking and updates
// to an existing supplier's configuration.
//
// Important notes:
// - Service configuration changes take effect at the start of the next session
// - Stake changes are processed immediately with appropriate token transfers
// - The supplier staking fee is charged for each staking operation
//
// TODO_POST_MAINNET(@red-0ne): Update supplier staking documentation to remove the upstaking requirement and introduce the staking fee.
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
	// Create or update a supplier using the configuration in the msg provided.
	_, err := k.Keeper.StakeSupplier(ctx, logger, msg)
	if err != nil {
		return nil, err
	}

	isSuccessful = true
	return &suppliertypes.MsgStakeSupplierResponse{}, nil
}

// createSupplier creates a new supplier entity from the given message.
// The new supplier will be active starting from the next session to ensure
// deterministic supplier selection for sessions.
func (k Keeper) createSupplier(
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
		// This prevents mid-session disruption to the session hydration process, which could
		// otherwise cause unexpected eviction of existing suppliers due to:
		//   1. The enforced maximum number of suppliers per session
		//   2. The deterministic random selection algorithm for suppliers
		// Note: This differs from applications, which are part of the session existence
		// (i.e. a session doesn't exist until its corresponding application is created).
		// Which is not the case for suppliers.
		Services:                make([]*sharedtypes.SupplierServiceConfig, 0),
		ServiceConfigHistory:    make([]*sharedtypes.ServiceConfigUpdate, 0),
		UnstakeSessionEndHeight: sharedtypes.SupplierNotUnstaking,
	}

	// Store the service configurations details of the newly created supplier.
	// They will take effect at the start of the next session.
	for _, serviceConfig := range msg.Services {
		servicesUpdate := &sharedtypes.ServiceConfigUpdate{
			OperatorAddress: msg.OperatorAddress,
			Service:         serviceConfig,
			// The effective block height is the start of the next session.
			ActivationHeight: nextSessionStartHeight,
		}
		supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, servicesUpdate)
	}

	return supplier
}

// StakeSupplier stakes (or updates) the supplier according to the given msg by applying the following logic:
//   - the msg is validated
//   - if the supplier is not found, it is created (in memory) according to the valid msg
//   - if the supplier is found and is not unbonding, it is updated (in memory) according to the msg
//   - if the supplier is found and is unbonding, it is updated (in memory; and no longer unbonding)
//   - additional stake validation (e.g. min stake, etc.)
//   - EITHER any positive difference between the msg stake and any current stake is transferred
//     from the staking supplier's account, to the supplier module's accounts.
//   - OR any negative difference between the msg stake and any current stake is transferred
//     from the supplier module's account (stake escrow) to the staking supplier's account.
//   - the supplier staking fee is deducted from the staking supplier's account balance.
//   - the (new or updated) supplier is persisted.
//   - an EventSupplierStaked event is emitted.
func (k Keeper) StakeSupplier(
	ctx context.Context,
	logger cosmoslog.Logger,
	msg *suppliertypes.MsgStakeSupplier,
) (*sharedtypes.Supplier, error) {

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
		// Ensure that a stake amount is provided if the supplier is being created.
		if msg.Stake == nil {
			return nil, status.Error(
				codes.InvalidArgument,
				suppliertypes.ErrSupplierInvalidStake.Wrap("when staking a new supplier, the stake amount MUST be non-nil").Error(),
			)
		}

		supplierCurrentStake = sdk.NewInt64Coin(pocket.DenomuPOKT, 0)
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

		// Only the operator can change service configurations. This ensures that
		// the owner cannot inadvertently modify services they don't understand.
		if !msg.IsSigner(supplier.OperatorAddress) && len(msg.Services) > 0 {
			err = sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrap(
				"only the operator account is authorized to update the service configurations",
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

	if err = k.reconcileSupplierStakeDiff(ctx, msg, supplierCurrentStake); err != nil {
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
	k.SetAndIndexDehydratedSupplier(ctx, supplier)
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
		OperatorAddress:  supplier.OperatorAddress,
		SessionEndHeight: sessionEndHeight,
	})
	if err = sdkCtx.EventManager().EmitTypedEvents(events...); err != nil {
		err = suppliertypes.ErrSupplierEmitEvent.Wrapf("(%+v): %s", events, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &supplier, nil
}

// updateSupplier updates an existing supplier with new configuration from the stake message.
// This includes updating the stake amount, owner address, and service configurations.
//
// Service configuration changes are scheduled to take effect at the next session start
// to ensure that current sessions remain stable and deterministic.
func (k Keeper) updateSupplier(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	msg *suppliertypes.MsgStakeSupplier,
) error {
	// If no stake amount is provided, preserve the current stake
	if msg.Stake == nil {
		msg.Stake = supplier.Stake
	}

	supplier.Stake = msg.Stake
	supplier.OwnerAddress = msg.OwnerAddress

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)

	updatedServiceConfigHistory := make([]*sharedtypes.ServiceConfigUpdate, 0)
	updatedServices := make(map[string]struct{})

	// Step 1: Add all new service configurations from the message
	//   - These configs will activate at the start of the next session
	//   - Track service IDs to identify which old inactive configs need replacement
	for _, newServiceConfig := range msg.Services {
		newServiceConfigUpdate := &sharedtypes.ServiceConfigUpdate{
			OperatorAddress: msg.OperatorAddress,
			Service:         newServiceConfig,
			// The effective block height is the start of the next session.
			ActivationHeight: nextSessionStartHeight,
		}
		updatedServiceConfigHistory = append(updatedServiceConfigHistory, newServiceConfigUpdate)
		updatedServices[newServiceConfig.ServiceId] = struct{}{}
	}

	// Step 2: Handle existing service configurations
	if len(msg.Services) > 0 {
		for _, oldServiceConfigUpdate := range supplier.ServiceConfigHistory {
			// Determine if this old config should be replaced vs deactivated:
			//   1. Check if this config would normally activate at next session
			//   2. Check if we have a new config for the same service
			//   3. If both true, skip it (new config already replaced it in Step 1)
			shouldActivateAtNextSession := oldServiceConfigUpdate.ActivationHeight == nextSessionStartHeight
			_, hasNewConfig := updatedServices[oldServiceConfigUpdate.Service.ServiceId]
			shouldReplaceWithNewConfig := shouldActivateAtNextSession && hasNewConfig

			if shouldReplaceWithNewConfig {
				continue
			}

			// Deactivate old configs that are NOT being replaced:
			//   - Currently active configs (no deactivation height set)
			//   - Configs scheduled to activate but for different services
			oldServiceConfigUpdate.DeactivationHeight = nextSessionStartHeight
			updatedServiceConfigHistory = append(updatedServiceConfigHistory, oldServiceConfigUpdate)
		}
	}

	// Step 3: Update the supplier with the final service configuration history
	supplier.ServiceConfigHistory = updatedServiceConfigHistory

	return nil
}

// reconcileSupplierStakeDiff transfers the difference between the current and new stake
// amounts by either escrowing, when the stake is increased, or unescrowing otherwise.
func (k Keeper) reconcileSupplierStakeDiff(
	ctx context.Context,
	msg *suppliertypes.MsgStakeSupplier,
	currentStake sdk.Coin,
) error {
	logger := k.Logger().With("method", "reconcileSupplierStakeDiff")

	newStake := *msg.Stake

	// Parse the signer address - this is the account that will pay for stake increases
	signerAccAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return err
	}

	// Parse the owner address - this is the account that will receive stake decreases
	ownerAccAddr, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		return err
	}

	// The Supplier is increasing its stake, so escrow the difference
	if currentStake.Amount.LT(newStake.Amount) {
		coinsToEscrow := sdk.NewCoins(newStake.Sub(currentStake))

		// Send the coins from the message signer account to the staked supplier pool
		return k.bankKeeper.SendCoinsFromAccountToModule(ctx, signerAccAddr, suppliertypes.ModuleName, coinsToEscrow)
	}

	// Ensure that the new stake is at least the minimum stake which is required for:
	// 1. The supplier to be considered active.
	// 2. Cover for any potential slashing that may occur during claims settlement.
	minStake := k.GetParams(ctx).MinStake
	if newStake.Amount.LT(minStake.Amount) {
		err := suppliertypes.ErrSupplierInvalidStake.Wrapf(
			"supplier with owner %q must stake at least %s",
			signerAccAddr, minStake,
		)
		return err
	}

	// The supplier is decreasing its stake, unescrow the difference.
	if currentStake.Amount.GT(newStake.Amount) {
		coinsToUnescrow := sdk.NewCoins(currentStake.Sub(newStake))

		// Send the coins from the staked supplier pool to the supplier owner account
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, suppliertypes.ModuleName, ownerAccAddr, coinsToUnescrow)
	}

	// The supplier is not changing its stake. This can happen if the supplier
	// is updating its service configurations or owner address but not the stake.
	logger.Info(fmt.Sprintf("Updating supplier with address %q but stake is unchanged", msg.OperatorAddress))
	return nil
}
