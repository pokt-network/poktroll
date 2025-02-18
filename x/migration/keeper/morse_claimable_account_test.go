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
		morseClaimableAccounts[i].Address = []byte(sample.MorseAddressHex())

		keeper.SetMorseClaimableAccount(ctx, morseClaimableAccounts[i])
	}
	return morseClaimableAccounts
}

func TestMorseClaimableAccountGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, item := range morseClaimableAccounts {
		rst, found := keeper.GetMorseClaimableAccount(ctx,
			item.Address.String(),
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
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	for _, item := range morseClaimableAccounts {
		keeper.RemoveMorseClaimableAccount(ctx,
			item.Address.String(),
		)
		_, found := keeper.GetMorseClaimableAccount(ctx,
			item.Address.String(),
		)
		require.False(t, found)
	}
}

func TestMorseClaimableAccountGetAll(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseClaimableAccounts := createNMorseClaimableAccount(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(morseClaimableAccounts),
		nullify.Fill(keeper.GetAllMorseClaimableAccount(ctx)),
	)
}
