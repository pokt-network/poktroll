package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNRelayMiningDifficulty(keeper keeper.Keeper, ctx context.Context, n int) []types.RelayMiningDifficulty {
	difficulties := make([]types.RelayMiningDifficulty, n)
	for idx := range difficulties {
		difficulties[idx].ServiceId = strconv.Itoa(idx)

		keeper.SetRelayMiningDifficulty(ctx, difficulties[idx])
	}
	return difficulties
}

func TestRelayMiningDifficultyGet(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	difficulties := createNRelayMiningDifficulty(keeper, ctx, 10)
	for _, difficulty := range difficulties {
		rst, found := keeper.GetRelayMiningDifficulty(ctx,
			difficulty.ServiceId,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&difficulty),
			nullify.Fill(&rst),
		)
	}
}
func TestRelayMiningDifficultyRemove(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	difficulties := createNRelayMiningDifficulty(keeper, ctx, 10)
	for _, difficulty := range difficulties {
		keeper.RemoveRelayMiningDifficulty(ctx,
			difficulty.ServiceId,
		)
		_, found := keeper.GetRelayMiningDifficulty(ctx,
			difficulty.ServiceId,
		)
		require.False(t, found)
	}
}

func TestRelayMiningDifficultyGetAll(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	difficulties := createNRelayMiningDifficulty(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(difficulties),
		nullify.Fill(keeper.GetAllRelayMiningDifficulty(ctx)),
	)
}
