package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNClaims(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Claim {
	claims := make([]types.Claim, n)
	for i := range claims {
		claims[i].SupplierAddress = sample.AccAddress()

		keeper.InsertClaim(ctx, claims[i])
	}
	return claims
}

func TestClaimGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	claims := createNClaims(keeper, ctx, 10)
	for _, claim := range claims {
		rst, found := keeper.GetClaim(ctx,
			claim.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&claim),
			nullify.Fill(&rst),
		)
	}
}
func TestClaimRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	claims := createNClaims(keeper, ctx, 10)
	for _, claim := range claims {
		keeper.RemoveClaim(ctx,
			claim.SessionId,
			claim.SupplierAddress,
		)
		_, found := keeper.GetClaim(ctx,
			claim.SessionId,
			claim.SupplierAddress,
		)
		require.False(t, found)
	}
}

func TestClaimGetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	claims := createNClaim(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(claims),
		nullify.Fill(keeper.GetAllClaims(ctx)),
	)
}
