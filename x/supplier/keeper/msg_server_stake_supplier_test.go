package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/supplier/keeper"
	"pocket/x/supplier/types"
)

func TestMsgServer_StakeSupplier_SuccessfulCreateAndUpdate(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the supplier
	addr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)

	// Prepare the supplier
	supplier := &types.MsgStakeSupplier{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
	}

	// Stake the supplier
	_, err := srv.StakeSupplier(wctx, supplier)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)
	require.Equal(t, addr, foundSupplier.Address)
	require.Equal(t, int64(100), foundSupplier.Stake.Amount.Int64())

	// Prepare an updated supplier with a higher stake
	updatedSupplier := &types.MsgStakeSupplier{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(200)},
	}

	// Update the staked supplier
	_, err = srv.StakeSupplier(wctx, updatedSupplier)
	require.NoError(t, err)
	foundSupplier, isSupplierFound = k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(200), foundSupplier.Stake.Amount.Int64())
}

func TestMsgServer_StakeSupplier_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Prepare the supplier
	addr := sample.AccAddress()
	supplier := &types.MsgStakeSupplier{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
	}

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(wctx, supplier)
	require.NoError(t, err)
	_, isSupplierFound := k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)

	// Prepare an updated supplier with a lower stake
	updatedSupplier := &types.MsgStakeSupplier{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(50)},
	}

	// Verify that it fails
	_, err = srv.StakeSupplier(wctx, updatedSupplier)
	require.Error(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(100), supplierFound.Stake.Amount.Int64())
}
