package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// UpdateParam updates a single parameter in the gateway module and returns
// all active parameters.
// * Validates the request message and authority permissions
// * Updates the specific parameter based on its name
// * Delegates to UpdateParams to handle validation and persistence
// * Returns both the current parameters and the scheduled parameter update
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *gatewaytypes.MsgUpdateParam,
) (*gatewaytypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	// Validate basic message structure and constraints
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	// Update the specific parameter based on its name
	switch msg.Name {
	case gatewaytypes.ParamMinStake:
		logger = logger.With("min_stake", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			gatewaytypes.ErrGatewayParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Create a full params update message and delegate to UpdateParams
	// This ensures:
	// * Authority validation
	// * Parameter constraints validation
	msgUpdateParams := &gatewaytypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, err
	}

	// Return a response with both the current parameters and the scheduled update
	// This allows the caller to see the current state and the scheduled change
	return &gatewaytypes.MsgUpdateParamResponse{
		Params:       response.Params,
		ParamsUpdate: response.ParamsUpdate,
	}, nil
}
