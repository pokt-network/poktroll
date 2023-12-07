package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgServer_UnstakeSupplier_Success(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t, nil)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the supplier
	addr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)

	// Prepare the supplier
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	stakeMsg := &types.MsgStakeSupplier{
		Address: addr,
		Stake:   &initialStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{
					Id: "svcId",
				},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     "http://localhost:8080",
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		},
	}

	// Stake the supplier
	_, err := srv.StakeSupplier(wctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)
	require.Equal(t, addr, foundSupplier.Address)
	require.Equal(t, initialStake.Amount, foundSupplier.Stake.Amount)
	require.Len(t, foundSupplier.Services, 1)

	// Unstake the supplier
	unstakeMsg := &types.MsgUnstakeSupplier{Address: addr}
	_, err = srv.UnstakeSupplier(wctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the supplier can no longer be found after unstaking
	_, isSupplierFound = k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_UnstakeSupplier_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t, nil)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the supplier
	addr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)

	// Unstake the supplier
	unstakeMsg := &types.MsgUnstakeSupplier{Address: addr}
	_, err := srv.UnstakeSupplier(wctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrSupplierNotFound)

	_, isSupplierFound = k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)
}
