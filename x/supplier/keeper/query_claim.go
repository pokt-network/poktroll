package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k Keeper) AllClaims(goCtx context.Context, req *types.QueryAllClaimsRequest) (*types.QueryAllClaimsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var claims []types.Claim
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	claimStore := prefix.NewStore(store, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

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

	return &types.QueryAllClaimsResponse{Claim: claims, Pagination: pageRes}, nil
}

func (k Keeper) Claim(goCtx context.Context, req *types.QueryGetClaimRequest) (*types.QueryGetClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetClaim(
		ctx,
		req.SessionId,
		req.SupplierAddress,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetClaimResponse{Claim: val}, nil
}
