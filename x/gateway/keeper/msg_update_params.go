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

// UpdateParams schedules a params update for the next session start height.
// It does not update the params immediately.
func (k msgServer) UpdateParams(
	goCtx context.Context,
	req *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if k.GetAuthority() != req.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrGatewayInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority,
			).Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := ctx.BlockHeight()

	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(sharedParamsUpdates, currentHeight)

	logger.Info(fmt.Sprintf(
		"About to schedule params update from [%v] to [%v] to be effective at block height %d",
		k.GetParams(ctx),
		req.Params,
		nextSessionStartHeight,
	))

	// If it is the first params are updated, then create a genesis params update
	// record with an effective block height of 1.
	// This is to keep track of the genesis params in history and as of when they
	// no longer became effective.
	paramsUpdates := k.GetParamsUpdates(ctx)
	if len(paramsUpdates) == 1 {
		params := k.GetParams(ctx)
		genesisParamsUpdate := types.ParamsUpdate{
			Params:               params,
			EffectiveBlockHeight: 1,
		}

		if err := k.SetParamsUpdate(ctx, genesisParamsUpdate); err != nil {
			err = types.ErrGatewayParamInvalid.Wrapf("unable to set params: %v", err)
			logger.Error(err.Error())
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// Do not directly update the params, instead, create a new params update object
	// and set it in the store. This will allow the new params to take effect at the
	// next session start height when the BeginBlockerActivateGatewayParams method is called.
	paramsUpdate := types.ParamsUpdate{
		Params:               req.Params,
		EffectiveBlockHeight: uint64(nextSessionStartHeight),
	}

	if err := k.SetParamsUpdate(ctx, paramsUpdate); err != nil {
		err = types.ErrGatewayParamInvalid.Wrapf("unable to set params: %v", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamsResponse{
		Params:               paramsUpdate.Params,
		EffectiveBlockHeight: paramsUpdate.EffectiveBlockHeight,
	}, nil
}
