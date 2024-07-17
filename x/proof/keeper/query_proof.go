package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k Keeper) AllProofs(ctx context.Context, req *proof.QueryAllProofsRequest) (*proof.QueryAllProofsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	var (
		// isCustomIndex is used to determined if we'll be using the store that points
		// to the actual Claim values, or a secondary index that points to the primary keys.
		isCustomIndex bool
		keyPrefix     []byte
	)

	switch filter := req.Filter.(type) {
	case *proof.QueryAllProofsRequest_SupplierAddress:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ProofSupplierAddressPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SupplierAddress)...)
	case *proof.QueryAllProofsRequest_SessionEndHeight:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ProofSessionEndHeightPrefix)
		keyPrefix = append(keyPrefix, []byte(fmt.Sprintf("%d", filter.SessionEndHeight))...)
	case *proof.QueryAllProofsRequest_SessionId:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ProofPrimaryKeyPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SessionId)...)
	default:
		isCustomIndex = false
		keyPrefix = types.KeyPrefix(types.ProofPrimaryKeyPrefix)
	}
	proofStore := prefix.NewStore(store, keyPrefix)

	var proofs []proof.Proof
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
			var proof proof.Proof
			if err := k.cdc.Unmarshal(value, &proof); err != nil {
				return err
			}

			proofs = append(proofs, proof)
		}

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proof.QueryAllProofsResponse{Proofs: proofs, Pagination: pageRes}, nil
}

func (k Keeper) Proof(ctx context.Context, req *proof.QueryGetProofRequest) (*proof.QueryGetProofResponse, error) {
	if req == nil {
		err := proof.ErrProofInvalidQueryRequest.Wrap("request cannot be nil")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	foundProof, isProofFound := k.GetProof(ctx, req.GetSessionId(), req.GetSupplierAddress())
	if !isProofFound {
		err := proof.ErrProofProofNotFound.Wrapf("session ID %q and supplier %q", req.SessionId, req.SupplierAddress)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &proof.QueryGetProofResponse{Proof: foundProof}, nil
}
