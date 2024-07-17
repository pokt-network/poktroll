package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/application"
)

func (k msgServer) UpdateParams(
	ctx context.Context,
	req *application.MsgUpdateParams,
) (*application.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}
	if k.GetAuthority() != req.Authority {
		return nil, application.ErrAppInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &application.MsgUpdateParamsResponse{}, nil
}
