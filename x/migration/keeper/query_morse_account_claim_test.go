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
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMorseAccountClaimQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	msgs := createNMorseAccountClaim(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetMorseAccountClaimRequest
		response *types.QueryGetMorseAccountClaimResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetMorseAccountClaimRequest{
				MorseSrcAddress: msgs[0].MorseSrcAddress,
			},
			response: &types.QueryGetMorseAccountClaimResponse{MorseAccountClaim: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetMorseAccountClaimRequest{
				MorseSrcAddress: msgs[1].MorseSrcAddress,
			},
			response: &types.QueryGetMorseAccountClaimResponse{MorseAccountClaim: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetMorseAccountClaimRequest{
				MorseSrcAddress: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.MorseAccountClaim(ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
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

func TestMorseAccountClaimQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	msgs := createNMorseAccountClaim(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllMorseAccountClaimRequest {
		return &types.QueryAllMorseAccountClaimRequest{
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
			resp, err := keeper.MorseAccountClaimAll(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.MorseAccountClaim), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.MorseAccountClaim),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.MorseAccountClaimAll(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.MorseAccountClaim), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.MorseAccountClaim),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.MorseAccountClaimAll(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.MorseAccountClaim),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.MorseAccountClaimAll(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
