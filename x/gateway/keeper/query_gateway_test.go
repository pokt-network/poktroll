package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestGatewayQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	gateways := createNGateways(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *gateway.QueryGetGatewayRequest
		response    *gateway.QueryGetGatewayResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &gateway.QueryGetGatewayRequest{
				Address: gateways[0].Address,
			},
			response: &gateway.QueryGetGatewayResponse{Gateway: gateways[0]},
		},
		{
			desc: "Second",
			request: &gateway.QueryGetGatewayRequest{
				Address: gateways[1].Address,
			},
			response: &gateway.QueryGetGatewayResponse{Gateway: gateways[1]},
		},
		{
			desc: "KeyNotFound",
			request: &gateway.QueryGetGatewayRequest{
				Address: strconv.Itoa(100000),
			},
			expectedErr: status.Error(codes.NotFound, fmt.Sprintf("gateway not found: address %s", strconv.Itoa(100000))),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Gateway(ctx, test.request)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(test.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestGatewayQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	gateways := createNGateways(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *gateway.QueryAllGatewaysRequest {
		return &gateway.QueryAllGatewaysRequest{
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
		for i := 0; i < len(gateways); i += step {
			resp, err := keeper.AllGateways(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Gateways), step)
			require.Subset(t,
				nullify.Fill(gateways),
				nullify.Fill(resp.Gateways),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(gateways); i += step {
			resp, err := keeper.AllGateways(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Gateways), step)
			require.Subset(t,
				nullify.Fill(gateways),
				nullify.Fill(resp.Gateways),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllGateways(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(gateways), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(gateways),
			nullify.Fill(resp.Gateways),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllGateways(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
