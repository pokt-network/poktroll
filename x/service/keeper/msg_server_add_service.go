package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

func (k msgServer) AddService(goCtx context.Context, msg *types.MsgAddService) (*types.MsgAddServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "AddService")
	logger.Info(fmt.Sprintf("About to add a new service with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Adding service failed basic validation: %v", err))
		return nil, err
	}

	return &types.MsgAddServiceResponse{}, nil
}
