package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestGetParams(t *testing.T) {
	k, ctx, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	// TODO_TECHDEBT(@bryanchriswhite, #394): Params tests don't assert initial state.
	params := tokenomics.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.ComputeUnitsToTokensMultiplier, k.ComputeUnitsToTokensMultiplier(ctx))
}

func TestParamsQuery(t *testing.T) {
	keeper, ctx, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	params := tokenomics.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &tokenomics.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &tokenomics.QueryParamsResponse{Params: params}, response)
}
