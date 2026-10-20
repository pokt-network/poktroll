package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz := store.Get(types.ParamsKey)
	if paramsBz == nil {
		return params
	}

	k.cdc.MustUnmarshal(paramsBz, &params)
	return params
}

// validateDaoRewardAddressCanReceiveFunds returns an error if the bank module
// refuses transfers to the DAO reward address. Settlement sends the DAO share,
// and every payout whose recipient cannot receive funds, to this address.
// The address format is checked by params validation, not here.
func (k Keeper) validateDaoRewardAddressCanReceiveFunds(params types.Params) error {
	daoAddr, err := cosmostypes.AccAddressFromBech32(params.GetDaoRewardAddress())
	if err != nil {
		return nil
	}
	if k.bankKeeper.BlockedAddr(daoAddr) {
		return types.ErrTokenomicsParamInvalid.Wrapf(
			"dao reward address %q cannot receive funds", params.GetDaoRewardAddress(),
		)
	}
	return nil
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

	return nil
}
