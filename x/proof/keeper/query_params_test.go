package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/proof"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	params := proof.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &proof.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &proof.QueryParamsResponse{Params: params}, response)
}
