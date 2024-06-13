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
	case types.ParamMinRelayDifficultyBits:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		minRelayDifficultyBits := uint64(value.AsInt64)

		if err := types.ValidateMinRelayDifficultyBits(minRelayDifficultyBits); err != nil {
			return nil, err
		}

		params.MinRelayDifficultyBits = minRelayDifficultyBits
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
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		proofRequirementThreshold := uint64(value.AsInt64)

		if err := types.ValidateProofRequirementThreshold(proofRequirementThreshold); err != nil {
			return nil, err
		}

		params.ProofRequirementThreshold = proofRequirementThreshold
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
