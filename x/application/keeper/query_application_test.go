package keeper_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/pokt-network/pocket/x/application/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestApplicationQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	msgs := createNApplications(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *types.QueryGetApplicationRequest
		response    *types.QueryGetApplicationResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetApplicationRequest{
				Address: msgs[0].Address,
			},
			response: &types.QueryGetApplicationResponse{Application: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetApplicationRequest{
				Address: msgs[1].Address,
			},
			response: &types.QueryGetApplicationResponse{Application: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetApplicationRequest{
				Address: strconv.Itoa(100000),
			},
			expectedErr: status.Error(
				codes.NotFound,
				types.ErrAppNotFound.Wrapf(
					"app address: %s",
					strconv.Itoa(100000),
				).Error(),
			),
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
				require.EqualError(t, err, test.expectedErr.Error())
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

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllApplicationsRequest {
		return &types.QueryAllApplicationsRequest{
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

func TestAllApplicationsQuery_WithDelegateeGatewayAddressConstraint(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	gatewayAddr1 := sample.AccAddress()
	appsWithDelegationAddr := []string{"1", "2"}
	apps := createNApplications(keeper, ctx, 5, withAppDelegateeGatewayAddr(gatewayAddr1, appsWithDelegationAddr))

	requestBuilder := func(gatewayAddr string) *types.QueryAllApplicationsRequest {
		return &types.QueryAllApplicationsRequest{
			DelegateeGatewayAddress: gatewayAddr,
		}
	}

	t.Run("QueryAppsWithDelegatee", func(t *testing.T) {
		resp, err := keeper.AllApplications(ctx, requestBuilder(gatewayAddr1))
		require.NoError(t, err)

		var expectedApps []types.Application
		for _, app := range apps {
			if slices.Contains(appsWithDelegationAddr, app.Address) {
				expectedApps = append(expectedApps, app)
			}
		}

		require.ElementsMatch(t,
			nullify.Fill(expectedApps),
			nullify.Fill(resp.Applications),
		)
	})

	t.Run("QueryAppsWithNoDelegationConstraint", func(t *testing.T) {
		resp, err := keeper.AllApplications(ctx, &types.QueryAllApplicationsRequest{})
		require.NoError(t, err)

		require.ElementsMatch(t,
			nullify.Fill(apps),
			nullify.Fill(resp.Applications),
		)
	})

	t.Run("QueryAppsWithInvalidGatewayAddr", func(t *testing.T) {
		addrInvalid := "invalid-address"
		_, err := keeper.AllApplications(ctx, requestBuilder(addrInvalid))
		require.ErrorContains(t, err, types.ErrQueryAppsInvalidGatewayAddress.Error())
	})
}
