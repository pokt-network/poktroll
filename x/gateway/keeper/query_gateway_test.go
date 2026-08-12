package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestGatewayQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	gateways := createNGateways(keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *types.QueryGetGatewayRequest
		response    *types.QueryGetGatewayResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetGatewayRequest{
				Address: gateways[0].Address,
			},
			response: &types.QueryGetGatewayResponse{Gateway: gateways[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetGatewayRequest{
				Address: gateways[1].Address,
			},
			response: &types.QueryGetGatewayResponse{Gateway: gateways[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetGatewayRequest{
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

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGatewaysRequest {
		return &types.QueryAllGatewaysRequest{
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

// TestGatewayQueryDehydrated verifies the `dehydrated` flag on both gateway queries.
//
// The flag exists because a card can be up to MaxServiceMetadataSizeBytes (256 KiB), so a
// default page of 100 carried cards is large enough to exceed the 4 MB default gRPC message
// limit. It defaults to FALSE (card included) to mirror the service queries -- a proto3
// bool cannot default to true.
func TestGatewayQueryDehydrated(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)

	gateways := createNGateways(keeper, ctx, 2)
	card := []byte(`{"schema":"pocket-gateway-card/v1","name":"test gateway"}`)
	for i := range gateways {
		gateways[i].Metadata = &sharedtypes.Metadata{Card: card}
		keeper.SetGateway(ctx, gateways[i])
	}

	t.Run("SingleHydratedByDefault", func(t *testing.T) {
		resp, err := keeper.Gateway(ctx, &types.QueryGetGatewayRequest{Address: gateways[0].Address})
		require.NoError(t, err)
		require.Equal(t, card, resp.Gateway.GetMetadata().GetCard())
	})

	t.Run("SingleDehydratedStripsCard", func(t *testing.T) {
		resp, err := keeper.Gateway(ctx, &types.QueryGetGatewayRequest{
			Address:    gateways[0].Address,
			Dehydrated: true,
		})
		require.NoError(t, err)
		require.Nil(t, resp.Gateway.Metadata)
		// Everything other than the card MUST survive dehydration.
		require.Equal(t, gateways[0].Address, resp.Gateway.Address)
		require.Equal(t, gateways[0].Stake, resp.Gateway.Stake)
	})

	t.Run("AllHydratedByDefault", func(t *testing.T) {
		resp, err := keeper.AllGateways(ctx, &types.QueryAllGatewaysRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Gateways, len(gateways))
		for _, gateway := range resp.Gateways {
			require.Equal(t, card, gateway.GetMetadata().GetCard())
		}
	})

	t.Run("AllDehydratedStripsEveryCard", func(t *testing.T) {
		resp, err := keeper.AllGateways(ctx, &types.QueryAllGatewaysRequest{Dehydrated: true})
		require.NoError(t, err)
		require.Len(t, resp.Gateways, len(gateways))
		for _, gateway := range resp.Gateways {
			require.Nil(t, gateway.Metadata)
			require.NotEmpty(t, gateway.Address)
		}
	})

	t.Run("DehydrationDoesNotMutateState", func(t *testing.T) {
		_, err := keeper.AllGateways(ctx, &types.QueryAllGatewaysRequest{Dehydrated: true})
		require.NoError(t, err)

		// The stored record MUST still carry the card: stripping is a response-shaping
		// concern, and the query keeper unmarshals into a local copy.
		stored, found := keeper.GetGateway(ctx, gateways[0].Address)
		require.True(t, found)
		require.Equal(t, card, stored.GetMetadata().GetCard())
	})
}
