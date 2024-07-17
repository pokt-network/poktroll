package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/session"
)

func (k msgServer) UpdateParams(ctx context.Context, req *session.MsgUpdateParams) (*session.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != req.Authority {
		return nil, session.ErrSessionInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &session.MsgUpdateParamsResponse{}, nil
}
