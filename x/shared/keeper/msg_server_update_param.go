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

		params.NumBlocksPerSession = uint64(value.AsInt64)
	case types.ParamGracePeriodEndOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.GracePeriodEndOffsetBlocks = uint64(value.AsInt64)
	case types.ParamClaimWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ClaimWindowOpenOffsetBlocks = uint64(value.AsInt64)
	case types.ParamClaimWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ClaimWindowCloseOffsetBlocks = uint64(value.AsInt64)
	case types.ParamProofWindowOpenOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofWindowOpenOffsetBlocks = uint64(value.AsInt64)
	case types.ParamProofWindowCloseOffsetBlocks:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ProofWindowCloseOffsetBlocks = uint64(value.AsInt64)
	case types.ParamSupplierUnbondingPeriodSessions:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.SupplierUnbondingPeriodSessions = uint64(value.AsInt64)
	case types.ParamApplicationUnbondingPeriodSessions:
		value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
		if !ok {
			return nil, types.ErrSharedParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}

		params.ApplicationUnbondingPeriodSessions = uint64(value.AsInt64)
	default:
		return nil, types.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
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
