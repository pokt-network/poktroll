package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) UpdateParams(
	ctx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrSharedInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("About to update params from [%v] to [%v]", k.GetParams(ctx), msg.Params))

	// Record the new params in history at their effective height (start of the next session)
	// and apply the live write per the narrow Option B rule (#543 anchored grid).
	// recordParamsHistory stamps the derived anchored-grid fields, overwriting any
	// governance-supplied anchor/number (§3.3). A num_blocks_per_session change is deferred to
	// the EndBlocker so in-flight sessions keep the old N; any other param takes effect on
	// live immediately, as before.
	if err := k.recordParamsHistory(ctx, msg.Params); err != nil {
		err = fmt.Errorf("unable to record params history: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.Info("Done updating params")

	return &types.MsgUpdateParamsResponse{}, nil
}
