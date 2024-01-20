package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// AllProofs returns all proofs stored on-chain.
func (k Keeper) AllProofs(
	goCtx context.Context,
	req *types.QueryAllProofsRequest,
) (*types.QueryAllProofsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(k.storeKey)

	var (
		// isCustomIndex is used to determined if we'll be using the store that points
		// to the actual Claim values, or a secondary index that points to the primary keys.
		isCustomIndex bool
		keyPrefix     []byte
	)

	switch filter := req.Filter.(type) {
	case *types.QueryAllProofsRequest_SupplierAddress:
		isCustomIndex = true
		keyPrefix = types.KeyPrefix(types.ProofSupplierAddressPrefix)
		keyPrefix = append(keyPrefix, []byte(filter.SupplierAddress)...)
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
			// We retrieve the primaryKey, and need to query the actual proof before decoding it.
			proof, proofFound := k.getProofByPrimaryKey(ctx, value)
			if proofFound {
				proofs = append(proofs, proof)
			}
		} else {
			// The value is an encoded proof.
			var proof types.Proof
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

	return &types.QueryAllProofsResponse{Proof: proofs, Pagination: pageRes}, nil
}

// Proof returns a singular proof according to the request.
func (k Keeper) Proof(
	goCtx context.Context,
	req *types.QueryGetProofRequest,
) (*types.QueryGetProofResponse, error) {
	if req == nil {
		err := types.ErrSupplierInvalidQueryRequest.Wrap("request cannot be nil")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetProof(ctx, req.GetSessionId(), req.GetSupplierAddress())
	if !found {
		err := types.ErrSupplierProofNotFound.Wrapf("session ID %q and supplier %q", req.SessionId, req.SupplierAddress)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryGetProofResponse{Proof: val}, nil
}
