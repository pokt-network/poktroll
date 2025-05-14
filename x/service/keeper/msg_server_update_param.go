package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// UpdateParam updates a single parameter in the service module and returns
// all active parameters.
// * Validates the request message and authority permissions
// * Updates the specific parameter based on its name
// * Delegates to UpdateParams to handle validation and persistence
// * Returns both the current parameters and the scheduled parameter update
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *servicetypes.MsgUpdateParam,
) (*servicetypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	// Validate basic message structure and constraints
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get current parameters to apply the single parameter update
	params := k.GetParams(ctx)

	// Update the specific parameter based on its name
	switch msg.Name {
	case servicetypes.ParamAddServiceFee:
		logger = logger.With("add_service_fee", msg.GetAsCoin())
		params.AddServiceFee = msg.GetAsCoin()
	case servicetypes.ParamTargetNumRelays:
		logger = logger.With("target_num_relays", msg.GetAsUint64())
		params.TargetNumRelays = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			servicetypes.ErrServiceParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Create a full params update message and delegate to UpdateParams
	// This ensures:
	// * Authority validation
	// * Parameter constraints validation
	msgUpdateParams := &servicetypes.MsgUpdateParams{
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
	return &servicetypes.MsgUpdateParamResponse{
		Params:       response.Params,
		ParamsUpdate: response.ParamsUpdate,
	}, nil
}
