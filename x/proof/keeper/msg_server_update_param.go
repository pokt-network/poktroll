package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *types.MsgUpdateParam,
) (*types.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrProofInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamProofRequestProbability:
		logger = logger.With("param_value", msg.GetAsFloat())
		params.ProofRequestProbability = msg.GetAsFloat()
	case types.ParamProofRequirementThreshold:
		logger = logger.With("param_value", msg.GetAsCoin())
		params.ProofRequirementThreshold = msg.GetAsCoin()
	case types.ParamProofMissingPenalty:
		logger = logger.With("param_value", msg.GetAsCoin())
		params.ProofMissingPenalty = msg.GetAsCoin()
	case types.ParamProofSubmissionFee:
		logger = logger.With("param_value", msg.GetAsCoin())
		params.ProofSubmissionFee = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrProofParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	if err := params.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	updatedParams := k.GetParams(ctx)

	return &types.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
