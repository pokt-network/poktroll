package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
    "github.com/pokt-network/poktroll/testutil/nullify"
)

func createTestMorseAccountState(keeper keeper.Keeper, ctx context.Context) types.MorseAccountState {
	item := types.MorseAccountState{}
	keeper.SetMorseAccountState(ctx, item)
	return item
}

func TestMorseAccountStateGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	item := createTestMorseAccountState(keeper, ctx)
	rst, found := keeper.GetMorseAccountState(ctx)
    require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestMorseAccountStateRemove(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	createTestMorseAccountState(keeper, ctx)
	keeper.RemoveMorseAccountState(ctx)
    _, found := keeper.GetMorseAccountState(ctx)
    require.False(t, found)
}
