package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/supplier/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ClaimAll(ctx context.Context, req *types.QueryAllClaimRequest) (*types.QueryAllClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var claims []types.Claim

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	claimStore := prefix.NewStore(store, types.KeyPrefix(types.ClaimKeyPrefix))

	pageRes, err := query.Paginate(claimStore, req.Pagination, func(key []byte, value []byte) error {
		var claim types.Claim
		if err := k.cdc.Unmarshal(value, &claim); err != nil {
			return err
		}

		claims = append(claims, claim)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllClaimResponse{Claim: claims, Pagination: pageRes}, nil
}

func (k Keeper) Claim(ctx context.Context, req *types.QueryGetClaimRequest) (*types.QueryGetClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetClaim(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetClaimResponse{Claim: val}, nil
}
