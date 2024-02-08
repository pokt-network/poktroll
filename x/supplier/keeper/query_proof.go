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

func (k Keeper) ProofAll(ctx context.Context, req *types.QueryAllProofRequest) (*types.QueryAllProofResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var proofs []types.Proof

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	proofStore := prefix.NewStore(store, types.KeyPrefix(types.ProofKeyPrefix))

	pageRes, err := query.Paginate(proofStore, req.Pagination, func(key []byte, value []byte) error {
		var proof types.Proof
		if err := k.cdc.Unmarshal(value, &proof); err != nil {
			return err
		}

		proofs = append(proofs, proof)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProofResponse{Proof: proofs, Pagination: pageRes}, nil
}

func (k Keeper) Proof(ctx context.Context, req *types.QueryGetProofRequest) (*types.QueryGetProofResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetProof(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetProofResponse{Proof: val}, nil
}
