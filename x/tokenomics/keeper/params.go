package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.ComputeToTokensMultiplier(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// ComputeToTokensMultiplier returns the ComputeToTokensMultiplier param
func (k Keeper) ComputeToTokensMultiplier(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyComputeToTokensMultiplier, &res)
	return
}
