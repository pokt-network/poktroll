package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

// AddService handles MsgAddService and adds a service to the network storing
// it in the service keeper's store using the provided ID from the message.
func (k msgServer) AddService(
	goCtx context.Context,
	msg *types.MsgAddService,
) (*types.MsgAddServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "AddService")
	logger.Info(fmt.Sprintf("About to add a new service with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Adding service failed basic validation: %v", err))
		return nil, err
	}

	if _, found := k.GetService(ctx, msg.Service.Id); found {
		logger.Error(fmt.Sprintf("Service already exists: %v", msg.Service))
		return nil, types.ErrServiceAlreadyExists
	}

	k.SetService(ctx, msg.Service)

	return &types.MsgAddServiceResponse{}, nil
}
