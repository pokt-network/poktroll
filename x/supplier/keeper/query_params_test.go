package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/supplier"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	params := supplier.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &supplier.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &supplier.QueryParamsResponse{Params: params}, response)
}
