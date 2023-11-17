package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

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

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(k.storeKey)

	// isCustomIndex is used to determined if we'll be using the store that points
	// to the actual Claim values, or a secondary index that points to the primary keys.
	var isCustomIndex bool
	var keyPrefix []byte
	switch filter := req.Filter.(type) {
	case *types.QueryAllClaimsRequest_SupplierAddress:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ClaimSupplierAddressPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SupplierAddress)...)

	case *types.QueryAllClaimsRequest_SessionEndHeight:
		isCustomIndex = true
		heightBz := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBz, filter.SessionEndHeight)

		keyPrefix = types.KeyPrefix(types.ClaimSessionEndHeightPrefix)
		keyPrefix = append(keyPrefix, heightBz...)

	case *types.QueryAllClaimsRequest_SessionId:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ClaimPrimaryKeyPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SessionId)...)

	default:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ClaimPrimaryKeyPrefix)
	}
	claimStore := prefix.NewStore(store, keyPrefix)

	var claims []types.Claim
	pageRes, err := query.Paginate(claimStore, req.Pagination, func(key []byte, value []byte) error {
		if isCustomIndex {
			// We retrieve the primaryKey, and need to query the actual Claim before decoding it.
			claim, claimFound := k.getClaimByPrimaryKey(ctx, value)
			if claimFound {
				claims = append(claims, claim)
			}
		} else {
			// The value is an encoded Claim.
			var claim types.Claim
			if err := k.cdc.Unmarshal(value, &claim); err != nil {
				return err
			}
			claims = append(claims, claim)
		}

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
		return nil, status.Error(codes.NotFound, fmt.Sprintf("claim not found for session %s and supplier %s", req.SessionId, req.SupplierAddress))
	}

	return &types.QueryGetClaimResponse{Claim: val}, nil
}
