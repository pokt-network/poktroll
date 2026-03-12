package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/service/types"
)

// TransferService transfers ownership of a service to a new address.
// Only the current owner can initiate the transfer.
func (k msgServer) TransferService(
	goCtx context.Context,
	msg *types.MsgTransferService,
) (*types.MsgTransferServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "TransferService")
	logger.Info(fmt.Sprintf("About to transfer service %q from %q to %q", msg.ServiceId, msg.OwnerAddress, msg.NewOwnerAddress))

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Look up the service by ID.
	foundService, found := k.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("service %q not found", msg.ServiceId).Error(),
		)
	}

	// Verify the signer is the current owner.
	if foundService.OwnerAddress != msg.OwnerAddress {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrServiceUnauthorized.Wrapf(
				"signer %q is not the owner of service %q (owner: %q)",
				msg.OwnerAddress, msg.ServiceId, foundService.OwnerAddress,
			).Error(),
		)
	}

	// Transfer ownership.
	foundService.OwnerAddress = msg.NewOwnerAddress
	k.SetService(ctx, foundService)

	logger.Info(fmt.Sprintf("Successfully transferred service %q to %q", msg.ServiceId, msg.NewOwnerAddress))

	return &types.MsgTransferServiceResponse{}, nil
}
