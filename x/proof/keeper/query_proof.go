package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k Keeper) AllProofs(ctx context.Context, req *types.QueryAllProofsRequest) (*types.QueryAllProofsResponse, error) {
	logger := k.Logger().With("method", "AllProofs")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	var (
		// isCustomIndex is used to determined if we'll be using the store that points
		// to the actual claim values, or a secondary index that points to the primary keys.
		isCustomIndex bool
		keyPrefix     []byte
	)

	switch filter := req.Filter.(type) {
	case *types.QueryAllProofsRequest_SupplierOperatorAddress:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ProofSupplierOperatorAddressPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SupplierOperatorAddress)...)
	case *types.QueryAllProofsRequest_SessionEndHeight:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ProofSessionEndHeightPrefix)
		keyPrefix = append(keyPrefix, []byte(fmt.Sprintf("%d", filter.SessionEndHeight))...)
	case *types.QueryAllProofsRequest_SessionId:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ProofPrimaryKeyPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SessionId)...)
	default:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ProofPrimaryKeyPrefix)
	}
	proofStore := prefix.NewStore(store, keyPrefix)

	var proofs []types.Proof
	pageRes, err := query.Paginate(proofStore, req.Pagination, func(key []byte, value []byte) error {
		if isCustomIndex {
			// If a custom index is used, the value is a primaryKey.
			// Then we retrieve the proof using the given primaryKey.
			foundProof, isProofFound := k.getProofByPrimaryKey(ctx, value)
			if isProofFound {
				proofs = append(proofs, foundProof)
			}
		} else {
			// The value is the encoded proof.
			var proof types.Proof
			if err := k.cdc.Unmarshal(value, &proof); err != nil {
				err = fmt.Errorf("unable to unmarshal proof with key (hex): %x: %+v", key, err)
				logger.Error(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			proofs = append(proofs, proof)
		}

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProofsResponse{Proofs: proofs, Pagination: pageRes}, nil
}

func (k Keeper) Proof(ctx context.Context, req *types.QueryGetProofRequest) (*types.QueryGetProofResponse, error) {
	if req == nil {
		err := types.ErrProofInvalidQueryRequest.Wrap("request cannot be nil")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	foundProof, isProofFound := k.GetProof(ctx, req.GetSessionId(), req.GetSupplierOperatorAddress())
	if !isProofFound {
		err := types.ErrProofProofNotFound.Wrapf("session ID %q and supplier %q", req.SessionId, req.SupplierOperatorAddress)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryGetProofResponse{Proof: foundProof}, nil
}
