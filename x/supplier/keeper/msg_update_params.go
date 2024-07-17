package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/supplier"
)

func (k msgServer) UpdateParams(
	ctx context.Context,
	req *supplier.MsgUpdateParams,
) (*supplier.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}
	if k.GetAuthority() != req.Authority {
		return nil, supplier.ErrSupplierInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &supplier.MsgUpdateParamsResponse{}, nil
}
