package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/pokt-network/pocket/x/migration/keeper"
	"github.com/pokt-network/pocket/x/migration/types"
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
func TestMorseClaimableAccountRemove(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, item := range morseClaimableAccounts {
		keeper.RemoveMorseClaimableAccount(ctx,
			item.MorseSrcAddress,
		)
		_, found := keeper.GetMorseClaimableAccount(ctx,
			item.MorseSrcAddress,
		)
		require.False(t, found)
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
