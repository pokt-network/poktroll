package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/service"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	params := service.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &service.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &service.QueryParamsResponse{Params: params}, response)
}
