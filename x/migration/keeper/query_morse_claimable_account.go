package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// MorseClaimableAccountAll returns all MorseClaimableAccounts created as a result of MsgImportMorseClaimableAccounts.
func (k Keeper) MorseClaimableAccountAll(
	ctx context.Context,
	req *types.QueryAllMorseClaimableAccountRequest,
) (*types.QueryAllMorseClaimableAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var morseClaimableAccounts []types.MorseClaimableAccount

	macStore := k.getMorseClaimableAccountStore(ctx)
	pageRes, err := query.Paginate(macStore, req.Pagination, func(key []byte, value []byte) error {
		var morseClaimableAccount types.MorseClaimableAccount
		if err := k.cdc.Unmarshal(value, &morseClaimableAccount); err != nil {
			return fmt.Errorf("unmarshalling error: %w", err)
		}

		morseClaimableAccounts = append(morseClaimableAccounts, morseClaimableAccount)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("pagination error: %w", err).Error())
	}

	return &types.QueryAllMorseClaimableAccountResponse{MorseClaimableAccount: morseClaimableAccounts, Pagination: pageRes}, nil
}

// MorseClaimableAccount returns a MorseClaimableAccount by its hex-encoded Morse address.
func (k Keeper) MorseClaimableAccount(
	ctx context.Context,
	req *types.QueryMorseClaimableAccountRequest,
) (*types.QueryMorseClaimableAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	morseClaimableAccount, found := k.GetMorseClaimableAccount(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryMorseClaimableAccountResponse{MorseClaimableAccount: morseClaimableAccount}, nil
}
