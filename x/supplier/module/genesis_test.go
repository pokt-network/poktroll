package supplier_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	suppliermodule "github.com/pokt-network/poktroll/x/supplier/module"
)

func TestGenesis(t *testing.T) {
	genesisState := supplier.GenesisState{
		Params: supplier.DefaultParams(),
		SupplierList: []shared.Supplier{
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.SupplierServiceConfig{
					{
						Service: &shared.Service{
							Id: "svcId1",
						},
						Endpoints: []*shared.SupplierEndpoint{
							{
								Url:     "http://localhost:8081",
								RpcType: shared.RPCType_JSON_RPC,
								Configs: make([]*shared.ConfigOption, 0),
							},
						},
					},
				},
			},
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.SupplierServiceConfig{
					{
						Service: &shared.Service{
							Id: "svcId2",
						},
						Endpoints: []*shared.SupplierEndpoint{
							{
								Url:     "http://localhost:8082",
								RpcType: shared.RPCType_GRPC,
								Configs: make([]*shared.ConfigOption, 0),
							},
						},
					},
				},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SupplierKeeper(t)
	suppliermodule.InitGenesis(ctx, k, genesisState)
	got := suppliermodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SupplierList, got.SupplierList)
	// this line is used by starport scaffolding # genesis/test/assert
}
