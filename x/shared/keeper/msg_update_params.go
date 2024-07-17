package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/shared"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *shared.MsgUpdateParams) (*shared.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != req.Authority {
		return nil, sdkerrors.Wrapf(shared.ErrSharedInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &shared.MsgUpdateParamsResponse{}, nil
}
