package keeper_test

import (
	"strconv"
	"testing"

	"pocket/x/supplier/keeper"
	"pocket/x/supplier/types"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNProof(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Proof {
	items := make([]types.Proof, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)
        
		keeper.SetProof(ctx, items[i])
	}
	return items
}

func TestProofGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNProof(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetProof(ctx,
		    item.Index,
            
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestProofRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNProof(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveProof(ctx,
		    item.Index,
            
		)
		_, found := keeper.GetProof(ctx,
		    item.Index,
            
		)
		require.False(t, found)
	}
}

func TestProofGetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNProof(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllProof(ctx)),
	)
}
