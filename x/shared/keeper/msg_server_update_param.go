package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *types.MsgUpdateParam) (*types.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, types.ErrSharedInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamNumBlocksPerSession:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		numBlocksPerSession := uint64(value.AsInt64)

		if err := types.ValidateNumBlocksPerSession(numBlocksPerSession); err != nil {
			return nil, err
		}

		params.NumBlocksPerSession = numBlocksPerSession
	case types.ParamClaimWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowOpenOffsetBlocks := uint64(value.AsInt64)

		if err := types.ValidateClaimWindowOpenOffsetBlocks(claimWindowOpenOffsetBlocks); err != nil {
			return nil, err
		}

		params.ClaimWindowOpenOffsetBlocks = claimWindowOpenOffsetBlocks
	case types.ParamClaimWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowCloseOffsetBlocks := uint64(value.AsInt64)

		if err := types.ValidateClaimWindowCloseOffsetBlocks(claimWindowCloseOffsetBlocks); err != nil {
			return nil, err
		}

		params.ClaimWindowCloseOffsetBlocks = claimWindowCloseOffsetBlocks
	case types.ParamProofWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowOpenOffsetBlocks := uint64(value.AsInt64)

		if err := types.ValidateProofWindowOpenOffsetBlocks(claimWindowOpenOffsetBlocks); err != nil {
			return nil, err
		}

		params.ProofWindowOpenOffsetBlocks = claimWindowOpenOffsetBlocks
	case types.ParamProofWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowCloseOffsetBlocks := uint64(value.AsInt64)

		if err := types.ValidateProofWindowCloseOffsetBlocks(claimWindowCloseOffsetBlocks); err != nil {
			return nil, err
		}

		params.ProofWindowCloseOffsetBlocks = claimWindowCloseOffsetBlocks
	default:
		return nil, types.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
