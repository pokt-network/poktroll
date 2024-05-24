package keeper_test

import (
	"context"
	"strconv"
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNRelayMiningDifficulty(keeper keeper.Keeper, ctx context.Context, n int) []types.RelayMiningDifficulty {
	items := make([]types.RelayMiningDifficulty, n)
	for i := range items {
		items[i].ServiceId = strconv.Itoa(i)

		keeper.SetRelayMiningDifficulty(ctx, items[i])
	}
	return items
}

func TestRelayMiningDifficultyGet(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	items := createNRelayMiningDifficulty(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetRelayMiningDifficulty(ctx,
			item.ServiceId,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestRelayMiningDifficultyRemove(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	items := createNRelayMiningDifficulty(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveRelayMiningDifficulty(ctx,
			item.ServiceId,
		)
		_, found := keeper.GetRelayMiningDifficulty(ctx,
			item.ServiceId,
		)
		require.False(t, found)
	}
}

func TestRelayMiningDifficultyGetAll(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	items := createNRelayMiningDifficulty(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllRelayMiningDifficulty(ctx)),
	)
}
