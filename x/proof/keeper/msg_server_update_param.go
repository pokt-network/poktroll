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
		proofRequestProbability := value.AsFloat

		if err := types.ValidateProofRequestProbability(proofRequestProbability); err != nil {
			return nil, err
		}

		params.ProofRequestProbability = proofRequestProbability
	case types.ParamProofRequirementThreshold:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofRequirementThreshold := value.AsCoin

		if err := types.ValidateProofRequirementThreshold(proofRequirementThreshold); err != nil {
			return nil, err
		}

		params.ProofRequirementThreshold = proofRequirementThreshold
	case types.ParamProofMissingPenalty:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofMissingPenalty := value.AsCoin

		if err := types.ValidateProofMissingPenalty(proofMissingPenalty); err != nil {
			return nil, err
		}

		params.ProofMissingPenalty = proofMissingPenalty
	case types.ParamProofSubmissionFee:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofSubmissionFee := value.AsCoin

		if err := types.ValidateProofSubmissionFee(proofSubmissionFee); err != nil {
			return nil, err
		}

		params.ProofSubmissionFee = proofSubmissionFee
	default:
		return nil, types.ErrProofParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
