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

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			tokenomicstypes.ErrTokenomicsInvalidAuthoritySigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {

	// MintAllocationPercentages
	case tokenomicstypes.ParamMintAllocationPercentages:
		logger = logger.With("param_value", msg.GetAsMintAllocationPercentages())
		params.MintAllocationPercentages = *msg.GetAsMintAllocationPercentages()

	// DaoRewardAddress
	case tokenomicstypes.ParamDaoRewardAddress:
		logger = logger.With("param_value", msg.GetAsString())
		params.DaoRewardAddress = msg.GetAsString()

	// GlobalInflationPerClaim
	case tokenomicstypes.ParamGlobalInflationPerClaim:
		logger = logger.With("param_value", msg.GetAsFloat())
		params.GlobalInflationPerClaim = msg.GetAsFloat()

	// MintEqualsBurnClaimDistribution
	case tokenomicstypes.ParamMintEqualsBurnClaimDistribution:
		logger = logger.With("param_value", msg.GetAsMintEqualsBurnClaimDistribution())
		params.MintEqualsBurnClaimDistribution = *msg.GetAsMintEqualsBurnClaimDistribution()

	// Default
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	if err := params.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &tokenomicstypes.MsgUpdateParamResponse{}, nil
}
