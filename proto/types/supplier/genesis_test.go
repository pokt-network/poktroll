package supplier_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	types "github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))
	serviceConfig1 := &shared.SupplierServiceConfig{
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
	}
	serviceList1 := []*shared.SupplierServiceConfig{serviceConfig1}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))
	serviceConfig2 := &shared.SupplierServiceConfig{
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
	}
	serviceList2 := []*shared.SupplierServiceConfig{serviceConfig2}

	tests := []struct {
		desc     string
		genState *types.GenesisState
		isValid  bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			isValid:  true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{

				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &stake2,
						Services: serviceList2,
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "invalid - zero supplier stake",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - negative supplier stake",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to duplicated supplier address",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr1,
						Stake:    &stake2,
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to nil supplier stake",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    nil,
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to missing supplier stake",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						// Stake explicitly omitted
						Services: serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - missing services list",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						// Services explicitly omitted
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - empty services list",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &stake2,
						Services: []*shared.SupplierServiceConfig{},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - invalid URL",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						Services: []*shared.SupplierServiceConfig{
							{
								Service: &shared.Service{
									Id: "svcId1",
								},
								Endpoints: []*shared.SupplierEndpoint{
									{
										Url:     "invalid URL",
										RpcType: shared.RPCType_JSON_RPC,
										Configs: make([]*shared.ConfigOption, 0),
									},
								},
							},
						},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - invalid RPC Type",
			genState: &types.GenesisState{
				SupplierList: []shared.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						Services: []*shared.SupplierServiceConfig{
							{
								Service: &shared.Service{
									Id: "svcId1",
								},
								Endpoints: []*shared.SupplierEndpoint{
									{
										Url:     "http://localhost:8081",
										RpcType: shared.RPCType_UNKNOWN_RPC,
										Configs: make([]*shared.ConfigOption, 0),
									},
								},
							},
						},
					},
				},
			},
			isValid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.genState.Validate()
			if test.isValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
