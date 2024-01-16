package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// TODO_IN_THIS_PR: How do we validate this is the same address that signed the request?
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrTokenomicsAuthorityAddressIncorrect
	}

	prevParams := k.GetParams(ctx)
	logger.Info("About to update params from [%v] to [%v]", prevParams, msg.Params)
	k.SetParams(ctx, msg.Params)
	logger.Info("Done updating params")

	return &types.MsgUpdateParamsResponse{}, nil
}

// GetParams get all parameters as types.Params
func (k TokenomicsKeeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.ComputeUnitsToTokensMultiplier(ctx),
	)
}

// SetParams set the params
func (k TokenomicsKeeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// ComputeUnitsToTokensMultiplier returns the ComputeUnitsToTokensMultiplier param
func (k TokenomicsKeeper) ComputeUnitsToTokensMultiplier(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyComputeUnitsToTokensMultiplier, &res)
	return
}
