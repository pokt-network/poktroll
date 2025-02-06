package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNMorseAccountClaim(
	keeper keeper.Keeper,
	ctx context.Context,
	numMorseAccountClaims int,
) []types.MorseAccountClaim {
	morseAccountClaims := make([]types.MorseAccountClaim, numMorseAccountClaims)
	for morseAccountClaimIdx := range morseAccountClaims {
		morseAccountClaims[morseAccountClaimIdx].MorseSrcAddress = strconv.Itoa(morseAccountClaimIdx)

		keeper.SetMorseAccountClaim(ctx, morseAccountClaims[morseAccountClaimIdx])
	}
	return morseAccountClaims
}

func TestMorseAccountClaimGet(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseAccountClaims := createNMorseAccountClaim(keeper, ctx, 10)
	for _, morseAccountClaim := range morseAccountClaims {
		foundMorseAccountClaim, isFound := keeper.GetMorseAccountClaim(ctx,
			morseAccountClaim.MorseSrcAddress,
		)
		require.True(t, isFound)
		require.Equal(t,
			nullify.Fill(&morseAccountClaim),
			nullify.Fill(&foundMorseAccountClaim),
		)
	}
}

func TestMorseAccountClaimRemove(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseAccountClaims := createNMorseAccountClaim(keeper, ctx, 10)
	for _, morseAccountClaim := range morseAccountClaims {
		keeper.RemoveMorseAccountClaim(ctx,
			morseAccountClaim.MorseSrcAddress,
		)
		_, isFound := keeper.GetMorseAccountClaim(ctx,
			morseAccountClaim.MorseSrcAddress,
		)
		require.False(t, isFound)
	}
}

func TestMorseAccountClaimGetAll(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	morseAccountClaims := createNMorseAccountClaim(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(morseAccountClaims),
		nullify.Fill(keeper.GetAllMorseAccountClaim(ctx)),
	)
}
