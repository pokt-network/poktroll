package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/proof"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *proof.MsgUpdateParam,
) (*proof.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, proof.ErrProofInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case proof.ParamMinRelayDifficultyBits:
		value, ok := msg.AsType.(*proof.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, proof.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		minRelayDifficultyBits := uint64(value.AsInt64)

		if err := proof.ValidateMinRelayDifficultyBits(minRelayDifficultyBits); err != nil {
			return nil, err
		}

		params.MinRelayDifficultyBits = minRelayDifficultyBits
	case proof.ParamProofRequestProbability:
		value, ok := msg.AsType.(*proof.MsgUpdateParam_AsFloat)
		if !ok {
			return nil, proof.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofRequestProbability := value.AsFloat

		if err := proof.ValidateProofRequestProbability(proofRequestProbability); err != nil {
			return nil, err
		}

		params.ProofRequestProbability = proofRequestProbability
	case proof.ParamProofRequirementThreshold:
		value, ok := msg.AsType.(*proof.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, proof.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofRequirementThreshold := uint64(value.AsInt64)

		if err := proof.ValidateProofRequirementThreshold(proofRequirementThreshold); err != nil {
			return nil, err
		}

		params.ProofRequirementThreshold = proofRequirementThreshold
	case proof.ParamProofMissingPenalty:
		value, ok := msg.AsType.(*proof.MsgUpdateParam_AsCoin)
		if !ok {
			return nil, proof.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofMissingPenalty := value.AsCoin

		if err := proof.ValidateProofMissingPenalty(proofMissingPenalty); err != nil {
			return nil, err
		}

		params.ProofMissingPenalty = proofMissingPenalty
	default:
		return nil, proof.ErrProofParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &proof.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
