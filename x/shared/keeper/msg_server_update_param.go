package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/shared"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *shared.MsgUpdateParam) (*shared.MsgUpdateParamResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, shared.ErrSharedInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case shared.ParamNumBlocksPerSession:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		numBlocksPerSession := uint64(value.AsInt64)

		if err := shared.ValidateNumBlocksPerSession(numBlocksPerSession); err != nil {
			return nil, err
		}

		params.NumBlocksPerSession = numBlocksPerSession
	case shared.ParamGracePeriodEndOffsetBlocks:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		gracePeriodEndOffsetBlocks := uint64(value.AsInt64)

		if err := shared.ValidateGracePeriodEndOffsetBlocks(gracePeriodEndOffsetBlocks); err != nil {
			return nil, err
		}

		params.GracePeriodEndOffsetBlocks = gracePeriodEndOffsetBlocks
	case shared.ParamClaimWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowOpenOffsetBlocks := uint64(value.AsInt64)

		if err := shared.ValidateClaimWindowOpenOffsetBlocks(claimWindowOpenOffsetBlocks); err != nil {
			return nil, err
		}

		params.ClaimWindowOpenOffsetBlocks = claimWindowOpenOffsetBlocks
	case shared.ParamClaimWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowCloseOffsetBlocks := uint64(value.AsInt64)

		if err := shared.ValidateClaimWindowCloseOffsetBlocks(claimWindowCloseOffsetBlocks); err != nil {
			return nil, err
		}

		params.ClaimWindowCloseOffsetBlocks = claimWindowCloseOffsetBlocks
	case shared.ParamProofWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowOpenOffsetBlocks := uint64(value.AsInt64)

		if err := shared.ValidateProofWindowOpenOffsetBlocks(claimWindowOpenOffsetBlocks); err != nil {
			return nil, err
		}

		params.ProofWindowOpenOffsetBlocks = claimWindowOpenOffsetBlocks
	case shared.ParamProofWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*shared.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		claimWindowCloseOffsetBlocks := uint64(value.AsInt64)

		if err := shared.ValidateProofWindowCloseOffsetBlocks(claimWindowCloseOffsetBlocks); err != nil {
			return nil, err
		}

		params.ProofWindowCloseOffsetBlocks = claimWindowCloseOffsetBlocks
	default:
		return nil, shared.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	updatedParams := k.GetParams(ctx)
	return &shared.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
