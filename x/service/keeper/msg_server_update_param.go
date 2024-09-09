package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/service/types"
)

// UpdateParam updates a single parameter in the service module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *types.MsgUpdateParam,
) (*types.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, types.ErrServiceInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamAddServiceFee:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrServiceParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		addServiceFee := value.AsCoin

		if err := types.ValidateAddServiceFee(addServiceFee); err != nil {
			return nil, err
		}

		params.AddServiceFee = addServiceFee
	default:
		return nil, types.ErrServiceParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
