package supplier_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Please see `x/supplier/types/genesis_test.go` for extensive tests related to the validity of the genesis state.
func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		SupplierList: []sharedtypes.Supplier{
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id: "svcId1",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8081",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id: "svcId2",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8082",
								RpcType: sharedtypes.RPCType_GRPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SupplierKeeper(t)
	supplier.InitGenesis(ctx, *k, genesisState)
	got := supplier.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SupplierList, got.SupplierList)
	// this line is used by starport scaffolding # genesis/test/assert
}
