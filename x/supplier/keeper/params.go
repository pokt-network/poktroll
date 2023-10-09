package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"pocket/x/supplier/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.EarlietClaimSubmissionOffset(ctx),
		k.EarliestProofSubmissionOffset(ctx),
		k.LatestClaimSubmissionBlocksInterval(ctx),
		k.LatestProofSubmissionBlocksInterval(ctx),
		k.ClaimSubmissionBlocksWindow(ctx),
		k.ProofSubmissionBlocksWindow(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// EarlietClaimSubmissionOffset returns the EarlietClaimSubmissionOffset param
func (k Keeper) EarlietClaimSubmissionOffset(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyEarlietClaimSubmissionOffset, &res)
	return
}

// EarliestProofSubmissionOffset returns the EarliestProofSubmissionOffset param
func (k Keeper) EarliestProofSubmissionOffset(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyEarliestProofSubmissionOffset, &res)
	return
}

// LatestClaimSubmissionBlocksInterval returns the LatestClaimSubmissionBlocksInterval param
func (k Keeper) LatestClaimSubmissionBlocksInterval(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyLatestClaimSubmissionBlocksInterval, &res)
	return
}

// LatestProofSubmissionBlocksInterval returns the LatestProofSubmissionBlocksInterval param
func (k Keeper) LatestProofSubmissionBlocksInterval(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyLatestProofSubmissionBlocksInterval, &res)
	return
}

// ClaimSubmissionBlocksWindow returns the ClaimSubmissionBlocksWindow param
func (k Keeper) ClaimSubmissionBlocksWindow(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyClaimSubmissionBlocksWindow, &res)
	return
}

// ProofSubmissionBlocksWindow returns the ProofSubmissionBlocksWindow param
func (k Keeper) ProofSubmissionBlocksWindow(ctx sdk.Context) (res int32) {
	k.paramstore.Get(ctx, types.KeyProofSubmissionBlocksWindow, &res)
	return
}
