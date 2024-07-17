package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
)

func TestMsgServer_UnstakeSupplier_Success(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)

	// Prepare the supplier
	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &supplier.MsgStakeSupplier{
		Address: supplierAddr,
		Stake:   &initialStake,
		Services: []*shared.SupplierServiceConfig{
			{
				Service: &shared.Service{
					Id: "svcId",
				},
				Endpoints: []*shared.SupplierEndpoint{
					{
						Url:     "http://localhost:8080",
						RpcType: shared.RPCType_JSON_RPC,
						Configs: make([]*shared.ConfigOption, 0),
					},
				},
			},
		},
	}

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierAddr, foundSupplier.Address)
	require.Equal(t, initialStake.Amount, foundSupplier.Stake.Amount)
	require.Len(t, foundSupplier.Services, 1)

	// Unstake the supplier
	unstakeMsg := &supplier.MsgUnstakeSupplier{Address: supplierAddr}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the supplier can no longer be found after unstaking
	_, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_UnstakeSupplier_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)

	// Unstake the supplier
	unstakeMsg := &supplier.MsgUnstakeSupplier{Address: supplierAddr}
	_, err := srv.UnstakeSupplier(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, supplier.ErrSupplierNotFound)

	_, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)
}
