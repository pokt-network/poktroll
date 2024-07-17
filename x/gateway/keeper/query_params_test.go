package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testkeeper.GatewayKeeper(t)
	params := gateway.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &gateway.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &gateway.QueryParamsResponse{Params: params}, response)
}
