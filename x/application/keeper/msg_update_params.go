package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}
	if k.GetAuthority() != req.Authority {
		return nil, types.ErrAppInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
