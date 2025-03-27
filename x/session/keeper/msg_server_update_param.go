package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// UpdateParam updates a single parameter in the session module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *sessiontypes.MsgUpdateParam,
) (*sessiontypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case sessiontypes.ParamNumSuppliersPerSession:
		logger = logger.With("numb_suppliers_per_session", msg.GetAsUint64())
		params.NumSuppliersPerSession = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			sessiontypes.ErrSessionParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &sessiontypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &sessiontypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
