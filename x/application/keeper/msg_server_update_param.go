package keeper

import (
	"context"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *apptypes.MsgUpdateParam) (*apptypes.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, apptypes.ErrAppInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	// TODO_IMPROVE: Add a Uint64 asType instead of using int64 for uint64 params.
	case apptypes.ParamMaxDelegatedGateways:
		if _, ok := msg.AsType.(*apptypes.MsgUpdateParam_AsInt64); !ok {
			return nil, apptypes.ErrAppParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		maxDelegatedGateways := uint64(msg.GetAsInt64())

		if err := apptypes.ValidateMaxDelegatedGateways(maxDelegatedGateways); err != nil {
			return nil, apptypes.ErrAppParamInvalid.Wrapf("maxdelegegated_gateways (%d): %v", maxDelegatedGateways, err)
		}
		params.MaxDelegatedGateways = maxDelegatedGateways
	case apptypes.ParamMinStake:
		if _, ok := msg.AsType.(*apptypes.MsgUpdateParam_AsCoin); !ok {
			return nil, apptypes.ErrAppParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		minStake := msg.GetAsCoin()

		if err := apptypes.ValidateMinStake(minStake); err != nil {
			return nil, err
		}
		params.MinStake = minStake
	default:
		return nil, apptypes.ErrAppParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &apptypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
