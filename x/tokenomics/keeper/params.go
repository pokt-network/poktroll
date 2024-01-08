package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

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
