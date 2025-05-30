package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetupService adds a new service or updates an existing one on the network.
//   - If the service already exists, it updates the service details.
//   - If the service does not exist, it creates a new service entry and deducts the
//     service fee from the message signer's account, sending it to the service module account.
func (k msgServer) SetupService(
	goCtx context.Context,
	msg *types.MsgSetupService,
) (*types.MsgSetupServiceResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"setup_service",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "SetupService")
	logger.Info(fmt.Sprintf("About to setup a service with msg: %v", msg))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Service setup failed basic validation: %v", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the service already exists or not.
	foundService, found := k.GetService(ctx, msg.Service.Id)
	if found {
		if err := k.updateService(ctx, msg, foundService, logger); err != nil {
			logger.Error(fmt.Sprintf("Failed to update existing service %q: %v", msg.Service.Id, err))
			return nil, err
		}
	} else {
		if err := k.addService(ctx, msg, logger); err != nil {
			logger.Error(fmt.Sprintf("Failed to add service %q: %v", msg.Service.Id, err))
			return nil, err
		}
	}

	isSuccessful = true
	return &types.MsgSetupServiceResponse{
		Service: &msg.Service,
	}, nil
}

// updateService updates an existing service in the store if it already exists.
//   - It checks if the signer address in the message matches the existing service owner address.
//   - It updates the service details in the store with immediate effect.
//   - Updating the service does not require a fee deduction, as it is assumed that the
//     service owner has already paid the service fee when the service was initially added.
func (k msgServer) updateService(
	ctx sdk.Context,
	msg *types.MsgSetupService,
	foundService sharedtypes.Service,
	logger log.Logger,
) error {
	if foundService.OwnerAddress != msg.Signer {
		return status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidOwnerAddress.Wrapf(
				"existing owner address %q does not match the message signer address %q",
				foundService.OwnerAddress, msg.Signer,
			).Error(),
		)
	}

	// TODO_POST_MIGRATION: Implement compute units per relay history to enable
	// non-disruptive updates w.r.t. ongoing claims settlements.

	logger.Info(fmt.Sprintf("Updating service: %v to %v", foundService, msg.Service))
	k.SetService(ctx, msg.Service)

	return nil
}

// addService adds a new service to the store.
//   - It checks if the message signer has enough funds to pay for the service fee.
//   - It deducts the service fee from the signer's account and sends it to the service module account.
//   - It allows the service owner to be different from the message signer, enabling
//     delegation of service creation to another account.
func (k msgServer) addService(
	ctx sdk.Context,
	msg *types.MsgSetupService,
	logger log.Logger,
) error {
	// Retrieve the address of the actor adding the service.
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidAddress.Wrapf(
				"%s is not in Bech32 format", msg.Signer,
			).Error(),
		)
	}

	// Check the actor has sufficient funds to pay for the add service fee.
	signerBalance := k.bankKeeper.GetBalance(ctx, signer, pocket.DenomuPOKT)

	// Check the balance of upokt is enough to cover the AddServiceFee.
	// Check if the serviceOwnerBalance is valid before comparing, since the amount
	// can be nil if the account does not have upokt balance.
	addServiceFee := *k.GetParams(ctx).AddServiceFee
	if !signerBalance.IsValid() || signerBalance.IsLT(addServiceFee) {
		return status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has %s, but the service fee is %s",
				signerBalance, k.GetParams(ctx).AddServiceFee,
			).Error(),
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.NewCoins(addServiceFee)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, signer, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct service fee from actor's balance: %+v", err))
		return status.Error(
			codes.Internal,
			types.ErrServiceFailedToDeductFee.Wrapf(
				"account has %s, failed to deduct %s due to: %v",
				signerBalance, k.GetParams(ctx).AddServiceFee,
				err.Error(),
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("Adding service: %v", msg.Service))
	k.SetService(ctx, msg.Service)

	return nil
}
