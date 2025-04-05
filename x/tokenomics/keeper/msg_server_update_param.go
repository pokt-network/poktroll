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
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *tokenomicstypes.MsgUpdateParam,
) (*tokenomicstypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

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

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, err
	}

	return &tokenomicstypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
