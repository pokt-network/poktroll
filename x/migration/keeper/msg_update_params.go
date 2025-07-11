package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) UpdateParams(
	ctx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	logger.Info(fmt.Sprintf("About to update params from [%v] to [%v]", k.GetParams(ctx), msg.Params))

	if err := k.SetParams(ctx, msg.Params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.Info("Done updating params")

	return &types.MsgUpdateParamsResponse{}, nil
}
