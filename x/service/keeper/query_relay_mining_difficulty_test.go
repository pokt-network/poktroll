package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/service/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestRelayMiningDifficultyQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	msgs := createNRelayMiningDifficulty(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *types.QueryGetRelayMiningDifficultyRequest
		response    *types.QueryGetRelayMiningDifficultyResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetRelayMiningDifficultyRequest{
				ServiceId: msgs[0].ServiceId,
			},
			response: &types.QueryGetRelayMiningDifficultyResponse{RelayMiningDifficulty: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetRelayMiningDifficultyRequest{
				ServiceId: msgs[1].ServiceId,
			},
			response: &types.QueryGetRelayMiningDifficultyResponse{RelayMiningDifficulty: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetRelayMiningDifficultyRequest{
				ServiceId: strconv.Itoa(100000),
			},
			expectedErr: status.Error(codes.NotFound, "not found"),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.RelayMiningDifficulty(ctx, tc.request)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestRelayMiningDifficultyQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	msgs := createNRelayMiningDifficulty(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllRelayMiningDifficultyRequest {
		return &types.QueryAllRelayMiningDifficultyRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.RelayMiningDifficultyAll(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RelayMiningDifficulty), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.RelayMiningDifficulty),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.RelayMiningDifficultyAll(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RelayMiningDifficulty), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.RelayMiningDifficulty),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.RelayMiningDifficultyAll(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.RelayMiningDifficulty),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.RelayMiningDifficultyAll(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
