package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/gateway"
)

func (k msgServer) UpdateParams(
	goCtx context.Context,
	req *gateway.MsgUpdateParams,
) (*gateway.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}
	if k.GetAuthority() != req.Authority {
		return nil, gateway.ErrGatewayInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	// NOTE(#322): Omitted parameters will be set to their zero value.
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &gateway.MsgUpdateParamsResponse{}, nil
}
