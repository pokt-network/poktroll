package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/gateway/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestGatewayQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGateway(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetGatewayRequest
		response *types.QueryGetGatewayResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetGatewayRequest{
				Address: msgs[0].Address,
			},
			response: &types.QueryGetGatewayResponse{Gateway: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetGatewayRequest{
				Address: msgs[1].Address,
			},
			response: &types.QueryGetGatewayResponse{Gateway: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetGatewayRequest{
				Address: strconv.Itoa(100000),
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
			response, err := keeper.Gateway(wctx, tc.request)
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

func TestGatewayQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNGateway(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGatewayRequest {
		return &types.QueryAllGatewayRequest{
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
			resp, err := keeper.GatewayAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Gateway), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Gateway),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.GatewayAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Gateway), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Gateway),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.GatewayAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Gateway),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.GatewayAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
