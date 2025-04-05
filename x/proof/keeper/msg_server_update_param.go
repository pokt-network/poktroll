package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *prooftypes.MsgUpdateParam,
) (*prooftypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case prooftypes.ParamProofRequestProbability:
		logger = logger.With("proof_request_probability", msg.GetAsFloat())
		params.ProofRequestProbability = msg.GetAsFloat()
	case prooftypes.ParamProofRequirementThreshold:
		logger = logger.With("proof_requirement_threshold", msg.GetAsCoin())
		params.ProofRequirementThreshold = msg.GetAsCoin()
	case prooftypes.ParamProofMissingPenalty:
		logger = logger.With("proof_missing_penalty", msg.GetAsCoin())
		params.ProofMissingPenalty = msg.GetAsCoin()
	case prooftypes.ParamProofSubmissionFee:
		logger = logger.With("proof_submission_fee", msg.GetAsCoin())
		params.ProofSubmissionFee = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			prooftypes.ErrProofParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &prooftypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, err
	}

	return &prooftypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
