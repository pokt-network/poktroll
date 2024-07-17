package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/application"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	params := application.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &application.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &application.QueryParamsResponse{Params: params}, response)
}
