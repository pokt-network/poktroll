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

func (k Keeper) MorseAccountClaimAll(ctx context.Context, req *types.QueryAllMorseAccountClaimRequest) (*types.QueryAllMorseAccountClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var morseAccountClaims []types.MorseAccountClaim

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	morseAccountClaimStore := prefix.NewStore(store, types.KeyPrefix(types.MorseAccountClaimKeyPrefix))

	pageRes, err := query.Paginate(morseAccountClaimStore, req.Pagination, func(key []byte, value []byte) error {
		var morseAccountClaim types.MorseAccountClaim
		if err := k.cdc.Unmarshal(value, &morseAccountClaim); err != nil {
			return err
		}

		morseAccountClaims = append(morseAccountClaims, morseAccountClaim)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllMorseAccountClaimResponse{MorseAccountClaim: morseAccountClaims, Pagination: pageRes}, nil
}

func (k Keeper) MorseAccountClaim(ctx context.Context, req *types.QueryGetMorseAccountClaimRequest) (*types.QueryGetMorseAccountClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetMorseAccountClaim(
		ctx,
		req.MorseSrcAddress,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetMorseAccountClaimResponse{MorseAccountClaim: val}, nil
}
