package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/service/types"
)

// UpdateService updates an existing service on the network.
// The operation checks if the signer has enough funds (upokt) to pay the UpdateServiceFee
// and if they are the owner of the service being updated.
// If funds are insufficient or signer is not the owner, the service won't be updated.
// Otherwise, the fee is transferred from the signer to the service module's account,
// and the service will be updated onchain.
func (k msgServer) UpdateService(
	goCtx context.Context,
	msg *types.MsgUpdateService,
) (*types.MsgUpdateServiceResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"update_service",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "UpdateService")
	logger.Info(fmt.Sprintf("About to update service with msg: %v", msg))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Updating service failed basic validation: %v", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the service exists.
	foundService, found := k.GetService(ctx, msg.Service.Id)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf(
				"service with ID %q not found",
				msg.Service.Id,
			).Error(),
		)
	}

	// Verify that the signer is the owner of the existing service.
	if foundService.OwnerAddress != msg.OwnerAddress {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrServiceInvalidOwnerAddress.Wrapf(
				"only the service owner can update the service. Expected %q, got %q",
				foundService.OwnerAddress, msg.OwnerAddress,
			).Error(),
		)
	}

	// Retrieve the address of the service owner.
	serviceOwnerAddr, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidAddress.Wrapf(
				"%s is not in Bech32 format", msg.OwnerAddress,
			).Error(),
		)
	}

	// Check the actor has sufficient funds to pay for the update service fee.
	accCoins := k.bankKeeper.SpendableCoins(ctx, serviceOwnerAddr)
	if accCoins.Len() == 0 {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has no spendable coins",
			).Error(),
		)
	}

	// Check the balance of upokt is enough to cover the UpdateServiceFee.
	accBalance := accCoins.AmountOf("upokt")
	updateServiceFee := k.GetParams(ctx).UpdateServiceFee
	if accBalance.LTE(updateServiceFee.Amount) {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has %s, but the update service fee is %s",
				accBalance, k.GetParams(ctx).UpdateServiceFee,
			).Error(),
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.NewCoins(*updateServiceFee)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, serviceOwnerAddr, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct update service fee from actor's balance: %+v", err))
		return nil, status.Error(
			codes.Internal,
			types.ErrServiceFailedToDeductFee.Wrapf(
				"account has %s, failed to deduct %s",
				accBalance, k.GetParams(ctx).UpdateServiceFee,
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("Updating service: %v", msg.Service))
	k.SetService(ctx, msg.Service)

	isSuccessful = true
	return &types.MsgUpdateServiceResponse{}, nil
}