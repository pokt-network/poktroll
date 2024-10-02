package keeper

import (
	"context"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(ctx context.Context, msg *suppliertypes.MsgUpdateParam) (*suppliertypes.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, suppliertypes.ErrSupplierInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case suppliertypes.ParamMinStake:
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &suppliertypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
