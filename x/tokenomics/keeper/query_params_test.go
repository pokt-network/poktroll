package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	// TODO_INVESTIGATE(#394): Params tests don't assert initial state.
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.ComputeUnitsToTokensMultiplier, k.ComputeUnitsToTokensMultiplier(ctx))
}

func TestParamsQuery(t *testing.T) {
	keeper, ctx, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
