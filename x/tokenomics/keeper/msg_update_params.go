package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// UpdateParams schedules a params update for the next session start height.
// It does not update the params immediately.
func (k msgServer) UpdateParams(
	goCtx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger()

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if msg.Authority != k.GetAuthority() {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrTokenomicsInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(),
				msg.Authority,
			).Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	committedHeight := ctx.BlockHeight()

	// Get the next session start height based on the current shared params.
	// This is used to determine when the params update will take effect.
	currentSharedParams := k.sharedKeeper.GetParams(ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&currentSharedParams, committedHeight)

	logger.Info(fmt.Sprintf(
		"About to schedule params update from [%v] to [%v] to be effective at block height %d",
		k.GetParams(ctx),
		msg.Params,
		nextSessionStartHeight,
	))

	// Do not directly update the params, instead, create a new params update object
	// and set it in the store. This will allow the new params to take effect at the
	// next session start height when the BeginBlockerActivateTokenomicsParams method is called.
	paramsUpdate := types.ParamsUpdate{
		Params:               msg.Params,
		EffectiveBlockHeight: uint64(nextSessionStartHeight),
	}

	// Store the params update but leave the params unchanged.
	if err := k.SetParamsUpdate(ctx, paramsUpdate); err != nil {
		err = types.ErrTokenomicsParamInvalid.Wrapf("unable to set params: %v", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamsResponse{
		Params:               paramsUpdate.Params,
		EffectiveBlockHeight: paramsUpdate.EffectiveBlockHeight,
	}, nil
}
