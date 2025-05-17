package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// UpdateParams schedules parameters update for the next session start height
// instead of updating the parameters immediately.
// * Validates the request message and checks authority permissions
// * Creates a ParamsUpdate object for delayed activation to the next session
// * Stores the update to be activated by the BeginBlocker
func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	// Validate basic message structure and constraints
	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify that the caller has proper authority to update parameters
	if k.GetAuthority() != req.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrGatewayInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), req.Authority,
			).Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := ctx.BlockHeight()

	// Get current parameters to return in the response
	currentParams := k.GetParams(ctx)

	// Calculate when the parameter update should take effect (i.e. at the beginning of the next session)
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(sharedParamsUpdates, currentHeight)

	logger.Info(fmt.Sprintf(
		"About to schedule params update from [%v] to [%v] to be effective at block height %d",
		currentParams,
		req.Params,
		nextSessionStartHeight,
	))

	// Set the deactivation height for the current parameters update
	// This is done to ensure that the current parameters are deactivated at the next session start height
	// This is necessary to avoid having two active parameters at the same time
	paramsUpdates := k.GetParamsUpdates(ctx)
	for _, paramsUpdate := range paramsUpdates {
		if paramsUpdate.DeactivationHeight != 0 {
			continue
		}

		paramsUpdate.DeactivationHeight = nextSessionStartHeight
		if err := k.SetParamsUpdate(ctx, *paramsUpdate); err != nil {
			err = types.ErrGatewayParamInvalid.Wrapf("unable to set params deactivation height: %v", err)
			logger.Error(err.Error())
			return nil, status.Error(codes.Internal, err.Error())
		}

		// There MUST be only one active params update at a time, so we can
		// safely break after setting the deactivation height for the current params
		break
	}

	// Instead of directly updating parameters, schedule them for activation:
	// * Create a new params update object for delayed activation
	// * Store it to be activated at the next session start height
	// * The actual parameter update will be applied by BeginBlockerActivateGatewayParams
	paramsUpdate := types.ParamsUpdate{
		Params:             req.Params,
		ActivationHeight:   nextSessionStartHeight,
		DeactivationHeight: 0,
	}

	// Persist the scheduled parameter update
	if err := k.SetParamsUpdate(ctx, paramsUpdate); err != nil {
		err = types.ErrGatewayParamInvalid.Wrapf("unable to set params: %v", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Return a response with both the current parameters and the scheduled update
	// This allows the caller to see the current state and the scheduled change
	return &types.MsgUpdateParamsResponse{
		Params:       currentParams,
		ParamsUpdate: paramsUpdate,
	}, nil
}
