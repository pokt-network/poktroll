package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
)

// IsServicePermissioned follows the number of entries, not the write history:
// re-adding an entry or removing an absent one never moves the count.
func TestIsServicePermissioned_FollowsEntryCount(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	addr1, addr2 := sample.AccAddressBech32(), sample.AccAddressBech32()

	k.SetSupplierAllowlistEntry(ctx, "svc", addr1)
	k.SetSupplierAllowlistEntry(ctx, "svc", addr1) // no-op
	k.RemoveSupplierAllowlistEntry(ctx, "svc", addr1)
	require.False(t, k.IsServicePermissioned(ctx, "svc"), "a repeated add counts once")

	k.SetSupplierAllowlistEntry(ctx, "svc", addr1)
	k.SetSupplierAllowlistEntry(ctx, "svc", addr2)
	k.RemoveSupplierAllowlistEntry(ctx, "svc", sample.AccAddressBech32()) // absent: no-op
	k.RemoveSupplierAllowlistEntry(ctx, "svc", addr1)
	require.True(t, k.IsServicePermissioned(ctx, "svc"), "one entry left")
	k.RemoveSupplierAllowlistEntry(ctx, "svc", addr1) // absent again: no-op
	require.True(t, k.IsServicePermissioned(ctx, "svc"))
	k.RemoveSupplierAllowlistEntry(ctx, "svc", addr2)
	require.False(t, k.IsServicePermissioned(ctx, "svc"))
	require.Empty(t, k.GetAllSupplierAllowlistEntries(ctx))

	// Another service's entries never count for this one.
	k.SetSupplierAllowlistEntry(ctx, "svc2", addr1)
	require.False(t, k.IsServicePermissioned(ctx, "svc"))
	require.True(t, k.IsServicePermissioned(ctx, "svc2"))
}

// IsServicePermissioned is exactly one Has, permissioned or not, whatever the
// block wrote before.
func TestIsServicePermissioned_IsOnePointRead(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	// One cache for the whole "block", as between the txs of a block.
	blockCtx, _ := sdk.UnwrapSDKContext(ctx).CacheContext()
	k.SetSupplierAllowlistEntry(blockCtx, "svc", sample.AccAddressBech32())
	for i := 0; i < 100; i++ {
		addr := sample.AccAddressBech32()
		k.SetSupplierAllowlistEntry(blockCtx, "svc", addr)
		k.RemoveSupplierAllowlistEntry(blockCtx, "svc", addr)
	}

	for _, svc := range []string{"svc", "public"} {
		meteredCtx := blockCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.IsServicePermissioned(meteredCtx, svc)
		require.Equal(t, storetypes.KVGasConfig().HasCost, meteredCtx.GasMeter().GasConsumed(), svc)
	}
}
