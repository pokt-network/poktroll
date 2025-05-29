package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/service/types"
)

func (k msgServer) DeleteService(ctx context.Context, msg *types.MsgDeleteService) (*types.MsgDeleteServiceResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// TODO: Handle the message

	return &types.MsgDeleteServiceResponse{}, nil
}
