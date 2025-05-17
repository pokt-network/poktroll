package keeper

//
import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// UpdateParam updates a single parameter in the tokenomics module and returns
// all active parameters.
// * Validates the request message and authority permissions
// * Updates the specific parameter based on its name
// * Delegates to UpdateParams to handle validation and persistence
// * Returns both the current parameters and the scheduled parameter update
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *tokenomicstypes.MsgUpdateParam,
) (*tokenomicstypes.MsgUpdateParamResponse, error) {
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

	switch msg.Name {
	case tokenomicstypes.ParamMintAllocationPercentages:
		logger = logger.With("mint_allocation_percentages", msg.GetAsMintAllocationPercentages())
		params.MintAllocationPercentages = *msg.GetAsMintAllocationPercentages()
	case tokenomicstypes.ParamDaoRewardAddress:
		logger = logger.With("dao_reward_address", msg.GetAsString())
		params.DaoRewardAddress = msg.GetAsString()
	case tokenomicstypes.ParamGlobalInflationPerClaim:
		logger = logger.With("global_inflation_per_claim", msg.GetAsFloat())
		params.GlobalInflationPerClaim = msg.GetAsFloat()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Create a full params update message and delegate to UpdateParams
	// This ensures:
	// * Authority validation
	// * Parameter constraints validation
	msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
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
	return &tokenomicstypes.MsgUpdateParamResponse{
		Params:       response.Params,
		ParamsUpdate: response.ParamsUpdate,
	}, nil
}
