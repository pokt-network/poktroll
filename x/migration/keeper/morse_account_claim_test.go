package keeper_test

import (
	"context"
	"strconv"
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNMorseAccountClaim(keeper keeper.Keeper, ctx context.Context, n int) []types.MorseAccountClaim {
	items := make([]types.MorseAccountClaim, n)
	for i := range items {
		items[i].MorseSrcAddress = strconv.Itoa(i)

		keeper.SetMorseAccountClaim(ctx, items[i])
	}
	return items
}

func TestMorseAccountClaimGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseAccountClaim(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetMorseAccountClaim(ctx,
			item.MorseSrcAddress,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestMorseAccountClaimRemove(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseAccountClaim(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveMorseAccountClaim(ctx,
			item.MorseSrcAddress,
		)
		_, found := keeper.GetMorseAccountClaim(ctx,
			item.MorseSrcAddress,
		)
		require.False(t, found)
	}
}

func TestMorseAccountClaimGetAll(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseAccountClaim(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllMorseAccountClaim(ctx)),
	)
}
