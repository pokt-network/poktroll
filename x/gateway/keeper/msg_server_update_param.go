package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(ctx context.Context, msg *types.MsgUpdateParam) (*types.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, types.ErrGatewayInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamMinStake:
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, types.ErrGatewayParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)

	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
