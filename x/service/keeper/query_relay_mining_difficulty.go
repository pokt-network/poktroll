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
			err = fmt.Errorf("unable to unmarshal relayMiningDifficulty with key (hex): %x: %w", key, err)
			logger.Error(err.Error())
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

func (k Keeper) RelayMiningDifficultyAtHeight(
	ctx context.Context,
	req *types.QueryGetRelayMiningDifficultyAtHeightRequest,
) (*types.QueryGetRelayMiningDifficultyAtHeightResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service ID is required")
	}

	if req.BlockHeight < 0 {
		return nil, status.Error(codes.InvalidArgument, "block height must be non-negative")
	}

	_, serviceFound := k.GetService(ctx, req.ServiceId)
	if !serviceFound {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("serviceID: %s", req.ServiceId).Error(),
		)
	}

	difficulty, _ := k.GetRelayMiningDifficultyAtHeight(ctx, req.ServiceId, req.BlockHeight)

	return &types.QueryGetRelayMiningDifficultyAtHeightResponse{RelayMiningDifficulty: difficulty}, nil
}

func (k Keeper) RelayMiningDifficultyHistory(
	ctx context.Context,
	req *types.QueryRelayMiningDifficultyHistoryRequest,
) (*types.QueryRelayMiningDifficultyHistoryResponse, error) {
	logger := k.Logger().With("method", "RelayMiningDifficultyHistory")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service ID is required")
	}

	_, serviceFound := k.GetService(ctx, req.ServiceId)
	if !serviceFound {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("serviceID: %s", req.ServiceId).Error(),
		)
	}

	var history []types.RelayMiningDifficultyUpdate

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceHistoryPrefix := types.RelayMiningDifficultyHistoryKeyPrefixForService(req.ServiceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	pageRes, err := query.Paginate(historyStore, req.Pagination, func(key []byte, value []byte) error {
		var difficultyUpdate types.RelayMiningDifficultyUpdate
		if err := k.cdc.Unmarshal(value, &difficultyUpdate); err != nil {
			err = fmt.Errorf("unable to unmarshal relayMiningDifficultyUpdate with key (hex): %x: %w", key, err)
			logger.Error(err.Error())
			return status.Error(codes.Internal, err.Error())
		}

		history = append(history, difficultyUpdate)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRelayMiningDifficultyHistoryResponse{
		RelayMiningDifficultyHistory: history,
		Pagination:                   pageRes,
	}, nil
}
