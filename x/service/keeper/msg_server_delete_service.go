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

// DeleteService removes a service from the network.
// Only the service owner can delete their service.
// No fee is charged for deleting a service.
func (k msgServer) DeleteService(
	goCtx context.Context,
	msg *types.MsgDeleteService,
) (*types.MsgDeleteServiceResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"delete_service",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "DeleteService")
	logger.Info(fmt.Sprintf("About to delete service with msg: %v", msg))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Deleting service failed basic validation: %v", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the service exists.
	foundService, found := k.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf(
				"service with ID %q not found",
				msg.ServiceId,
			).Error(),
		)
	}

	// Verify that the signer is the owner of the service.
	if foundService.OwnerAddress != msg.OwnerAddress {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrServiceInvalidOwnerAddress.Wrapf(
				"only the service owner can delete the service. Expected %q, got %q",
				foundService.OwnerAddress, msg.OwnerAddress,
			).Error(),
		)
	}

	// Validate the owner address format.
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidAddress.Wrapf(
				"%s is not in Bech32 format", msg.OwnerAddress,
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("Deleting service with ID: %s", msg.ServiceId))
	k.RemoveService(ctx, msg.ServiceId)

	isSuccessful = true
	return &types.MsgDeleteServiceResponse{}, nil
}