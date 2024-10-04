package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *types.MsgUpdateParam,
) (*types.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, types.ErrProofInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamProofRequestProbability:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsFloat)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofRequestProbability = value.AsFloat
	case types.ParamProofRequirementThreshold:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofRequirementThreshold = value.AsCoin
	case types.ParamProofMissingPenalty:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofMissingPenalty = value.AsCoin
	case types.ParamProofSubmissionFee:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofSubmissionFee = value.AsCoin
	default:
		return nil, types.ErrProofParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := params.ValidateBasic(); err != nil {
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
