package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.SharedKeeper(t)
	params := shared.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &shared.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &shared.QueryParamsResponse{Params: params}, response)
}
