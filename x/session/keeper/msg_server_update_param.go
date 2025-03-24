package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"

	"google.golang.org/grpc/codes"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(ctx context.Context, msg *sessiontypes.MsgUpdateParam) (*sessiontypes.MsgUpdateParamResponse, error) {
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
			suppliertypes.ErrSupplierInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case sessiontypes.ParamNumSuppliersPerSession:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.NumSuppliersPerSession = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	updatedParams := k.GetParams(ctx)

	return &sessiontypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
