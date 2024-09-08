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

func TestServiceQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	msgs := createNServices(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *types.QueryGetServiceRequest
		response    *types.QueryGetServiceResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetServiceRequest{
				Id: msgs[0].Id,
			},
			response: &types.QueryGetServiceResponse{Service: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetServiceRequest{
				Id: msgs[1].Id,
			},
			response: &types.QueryGetServiceResponse{Service: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetServiceRequest{
				Id: strconv.Itoa(100000),
			},
			expectedErr: status.Error(codes.NotFound, "service ID not found"),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Service(ctx, test.request)
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

func TestServiceQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	msgs := createNServices(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllServicesRequest {
		return &types.QueryAllServicesRequest{
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
			resp, err := keeper.AllServices(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Service), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Service),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.AllServices(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Service), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Service),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllServices(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Service),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllServices(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
