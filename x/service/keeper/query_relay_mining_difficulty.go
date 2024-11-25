package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/service/types"
)

func (k Keeper) RelayMiningDifficultyAll(ctx context.Context, req *types.QueryAllRelayMiningDifficultyRequest) (*types.QueryAllRelayMiningDifficultyResponse, error) {
	logger := k.Logger().With("method", "RelayMiningDifficultyAll")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var relayMiningDifficulties []types.RelayMiningDifficulty

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	relayMiningDifficultyStore := prefix.NewStore(store, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	pageRes, err := query.Paginate(relayMiningDifficultyStore, req.Pagination, func(key []byte, value []byte) error {
		var relayMiningDifficulty types.RelayMiningDifficulty
		if err := k.cdc.Unmarshal(value, &relayMiningDifficulty); err != nil {
			logger.Error(fmt.Sprintf("unable to unmarshal relayMiningDifficulty with key (hex): %x: %+v", key, err))
			return status.Error(codes.Internal, err.Error())
		}

		relayMiningDifficulties = append(relayMiningDifficulties, relayMiningDifficulty)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRelayMiningDifficultyResponse{RelayMiningDifficulty: relayMiningDifficulties, Pagination: pageRes}, nil
}

func (k Keeper) RelayMiningDifficulty(ctx context.Context, req *types.QueryGetRelayMiningDifficultyRequest) (*types.QueryGetRelayMiningDifficultyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, serviceFound := k.GetService(ctx, req.ServiceId)
	if !serviceFound {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("serviceID: %s", req.ServiceId).Error(),
		)
	}

	difficulty, _ := k.GetRelayMiningDifficulty(ctx, req.ServiceId)

	return &types.QueryGetRelayMiningDifficultyResponse{RelayMiningDifficulty: difficulty}, nil
}
