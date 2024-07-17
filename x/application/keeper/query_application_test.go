package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/application"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestApplicationQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	msgs := createNApplications(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *application.QueryGetApplicationRequest
		response    *application.QueryGetApplicationResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &application.QueryGetApplicationRequest{
				Address: msgs[0].Address,
			},
			response: &application.QueryGetApplicationResponse{Application: msgs[0]},
		},
		{
			desc: "Second",
			request: &application.QueryGetApplicationRequest{
				Address: msgs[1].Address,
			},
			response: &application.QueryGetApplicationResponse{Application: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &application.QueryGetApplicationRequest{
				Address: strconv.Itoa(100000),
			},
			expectedErr: status.Error(codes.NotFound, "application not found"),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Application(ctx, test.request)
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

func TestApplicationQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *application.QueryAllApplicationsRequest {
		return &application.QueryAllApplicationsRequest{
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
		for i := 0; i < len(apps); i += step {
			resp, err := keeper.AllApplications(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Applications), step)
			require.Subset(t,
				nullify.Fill(apps),
				nullify.Fill(resp.Applications),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(apps); i += step {
			resp, err := keeper.AllApplications(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Applications), step)
			require.Subset(t,
				nullify.Fill(apps),
				nullify.Fill(resp.Applications),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllApplications(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(apps), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(apps),
			nullify.Fill(resp.Applications),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllApplications(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
