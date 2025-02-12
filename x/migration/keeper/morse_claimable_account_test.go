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

func createNMorseClaimableAccount(keeper keeper.Keeper, ctx context.Context, n int) []types.MorseClaimableAccount {
	items := make([]types.MorseClaimableAccount, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)

		keeper.SetMorseClaimableAccount(ctx, items[i])
	}
	return items
}

func TestMorseClaimableAccountGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetMorseClaimableAccount(ctx,
			item.Address,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestMorseClaimableAccountRemove(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveMorseClaimableAccount(ctx,
			item.Address,
		)
		_, found := keeper.GetMorseClaimableAccount(ctx,
			item.Address,
		)
		require.False(t, found)
	}
}

func TestMorseClaimableAccountGetAll(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	items := createNMorseClaimableAccount(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllMorseClaimableAccount(ctx)),
	)
}
