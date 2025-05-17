package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// UpdateParam updates a single parameter in the supplier module and returns
// all active parameters.
// * Validates the request message and authority permissions
// * Updates the specific parameter based on its name
// * Delegates to UpdateParams to handle validation and persistence
// * Returns both the current parameters and the scheduled parameter update
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *suppliertypes.MsgUpdateParam,
) (*suppliertypes.MsgUpdateParamResponse, error) {
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
	case suppliertypes.ParamMinStake:
		logger = logger.With("min_stake", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	case suppliertypes.ParamStakingFee:
		logger = logger.With("staking_fee", msg.GetAsCoin())
		params.StakingFee = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Create a full params update message and delegate to UpdateParams
	// This ensures:
	// * Authority validation
	// * Parameter constraints validation
	msgUpdateParams := &suppliertypes.MsgUpdateParams{
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
	return &suppliertypes.MsgUpdateParamResponse{
		Params:       response.Params,
		ParamsUpdate: response.ParamsUpdate,
	}, nil
}
