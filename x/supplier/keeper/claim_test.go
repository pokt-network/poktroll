package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNClaims(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Claim {
	items := make([]types.Claim, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetClaim(ctx, items[i])
	}
	return items
}

func TestClaimGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNClaims(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetClaim(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestClaimRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNClaims(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveClaim(ctx,
			item.Index,
		)
		_, found := keeper.GetClaim(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestGetAllClaims(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNClaims(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllClaims(ctx)),
	)
}
