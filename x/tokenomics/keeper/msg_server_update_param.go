package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *types.MsgUpdateParam) (*types.MsgUpdateParamResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, types.ErrTokenomicsInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case "compute_units_to_tokens_multiplier":
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, fmt.Errorf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ComputeUnitsToTokensMultiplier = uint64(value.AsInt64)
	default:
		return nil, fmt.Errorf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
