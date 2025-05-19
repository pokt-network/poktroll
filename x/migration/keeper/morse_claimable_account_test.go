package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNMorseClaimableAccount(keeper keeper.Keeper, ctx context.Context, n int) []types.MorseClaimableAccount {
	morseClaimableAccounts := make([]types.MorseClaimableAccount, n)
	for i := range morseClaimableAccounts {
		morseClaimableAccounts[i].MorseSrcAddress = sample.MorseAddressHex()

		keeper.SetMorseClaimableAccount(ctx, morseClaimableAccounts[i])
	}
	return morseClaimableAccounts
}

func TestMorseClaimableAccountGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, morseClaimableAccount := range morseClaimableAccounts {
		rst, found := keeper.GetMorseClaimableAccount(ctx,
			morseClaimableAccount.MorseSrcAddress,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&morseClaimableAccount),
			nullify.Fill(&rst),
		)
	}
}

func TestMorseClaimableAccountGetAll(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(morseClaimableAccounts),
		nullify.Fill(keeper.GetAllMorseClaimableAccounts(ctx)),
	)
}

func TestHasAnyMorseClaimableAccounts(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)

	// Keeper state should initially be empty (i.e. no MorseClaimableAccounts).
	require.False(t, keeper.HasAnyMorseClaimableAccounts(ctx))

	// Populate the keeper state with EXACTLY 1 MorseClaimableAccounts.
	_ = createNMorseClaimableAccount(keeper, ctx, 1)

	// Keeper state should cause HasAnyMorseClaimableAccounts to return true.
	require.True(t, keeper.HasAnyMorseClaimableAccounts(ctx))
}
