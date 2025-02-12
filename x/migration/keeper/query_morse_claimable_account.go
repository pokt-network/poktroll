package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/migration/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MorseClaimableAccountAll(ctx context.Context, req *types.QueryAllMorseClaimableAccountRequest) (*types.QueryAllMorseClaimableAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var morseClaimableAccounts []types.MorseClaimableAccount

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	morseClaimableAccountStore := prefix.NewStore(store, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))

	pageRes, err := query.Paginate(morseClaimableAccountStore, req.Pagination, func(key []byte, value []byte) error {
		var morseClaimableAccount types.MorseClaimableAccount
		if err := k.cdc.Unmarshal(value, &morseClaimableAccount); err != nil {
			return err
		}

		morseClaimableAccounts = append(morseClaimableAccounts, morseClaimableAccount)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllMorseClaimableAccountResponse{MorseClaimableAccount: morseClaimableAccounts, Pagination: pageRes}, nil
}

func (k Keeper) MorseClaimableAccount(ctx context.Context, req *types.QueryGetMorseClaimableAccountRequest) (*types.QueryGetMorseClaimableAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetMorseClaimableAccount(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetMorseClaimableAccountResponse{MorseClaimableAccount: val}, nil
}
