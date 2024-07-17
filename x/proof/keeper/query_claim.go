package keeper

import (
	"context"
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k Keeper) AllClaims(ctx context.Context, req *proof.QueryAllClaimsRequest) (*proof.QueryAllClaimsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	// isCustomIndex is used to determined if we'll be using the store that points
	// to the actual Claim values, or a secondary index that points to the primary keys.
	var (
		isCustomIndex bool
		keyPrefix     []byte
	)

	switch filter := req.Filter.(type) {
	case *proof.QueryAllClaimsRequest_SupplierAddress:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ClaimSupplierAddressPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SupplierAddress)...)

	case *proof.QueryAllClaimsRequest_SessionEndHeight:
		isCustomIndex = true
		heightBz := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBz, filter.SessionEndHeight)
		keyPrefix = types.KeyPrefix(types.ClaimSessionEndHeightPrefix)
		keyPrefix = append(keyPrefix, heightBz...)

	case *proof.QueryAllClaimsRequest_SessionId:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ClaimPrimaryKeyPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SessionId)...)

	default:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ClaimPrimaryKeyPrefix)
	}

	claimStore := prefix.NewStore(storeAdapter, keyPrefix)

	var claims []proof.Claim
	pageRes, err := query.Paginate(claimStore, req.Pagination, func(key []byte, value []byte) error {
		if isCustomIndex {
			// If a custom index is used, the value is a primaryKey.
			// Then we retrieve the claim using the given primaryKey.
			foundClaim, isClaimFound := k.getClaimByPrimaryKey(ctx, value)
			if isClaimFound {
				claims = append(claims, foundClaim)
			}
		} else {
			// The value is the encoded claim.
			var claim proof.Claim
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

	return &proof.QueryAllClaimsResponse{Claims: claims, Pagination: pageRes}, nil
}

func (k Keeper) Claim(ctx context.Context, req *proof.QueryGetClaimRequest) (*proof.QueryGetClaimResponse, error) {
	if req == nil {
		err := proof.ErrProofInvalidQueryRequest.Wrapf("request cannot be nil")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	foundClaim, isClaimFound := k.GetClaim(
		ctx,
		req.SessionId,
		req.SupplierAddress,
	)
	if !isClaimFound {
		err := proof.ErrProofClaimNotFound.Wrapf("session ID %q and supplier %q", req.SessionId, req.SupplierAddress)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &proof.QueryGetClaimResponse{Claim: foundClaim}, nil
}
