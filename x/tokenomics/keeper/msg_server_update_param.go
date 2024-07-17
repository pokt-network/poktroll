package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
)

// UpdateParam updates a single parameter in the tokenomics module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *tokenomics.MsgUpdateParam,
) (*tokenomics.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, tokenomics.ErrTokenomicsInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case tokenomics.ParamComputeUnitsToTokensMultiplier:
		value, ok := msg.AsType.(*tokenomics.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, tokenomics.ErrTokenomicsParamsInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		computeUnitsToTokensMultiplier := uint64(value.AsInt64)

		if err := tokenomics.ValidateComputeUnitsToTokensMultiplier(computeUnitsToTokensMultiplier); err != nil {
			return nil, err
		}

		params.ComputeUnitsToTokensMultiplier = computeUnitsToTokensMultiplier
	default:
		return nil, tokenomics.ErrTokenomicsParamsInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &tokenomics.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
