package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))
	serviceConfig1 := &sharedtypes.SupplierServiceConfig{
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
	}
	serviceList1 := []*sharedtypes.SupplierServiceConfig{serviceConfig1}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))
	serviceConfig2 := &sharedtypes.SupplierServiceConfig{
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
	}
	serviceList2 := []*sharedtypes.SupplierServiceConfig{serviceConfig2}

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{

				SupplierList: []sharedtypes.Supplier{
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
				ClaimList: []types.Claim{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - zero supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
						Services: serviceList2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
						Services: serviceList2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
						Services: serviceList2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
						Services: serviceList2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to duplicated supplier address",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
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
			valid: false,
		},
		{
			desc: "invalid - due to nil supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
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
			valid: false,
		},
		{
			desc: "invalid - due to missing supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						// Explicitly missing stake
						Services: serviceList2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing services list",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						// Services: intentionally omitted
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - empty services list",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address:  addr2,
						Stake:    &stake2,
						Services: []*sharedtypes.SupplierServiceConfig{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - invalid URL",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						Services: []*sharedtypes.SupplierServiceConfig{
							{
								Service: &sharedtypes.Service{
									Id: "svcId1",
								},
								Endpoints: []*sharedtypes.SupplierEndpoint{
									{
										Url:     "invalid URL",
										RpcType: sharedtypes.RPCType_JSON_RPC,
										Configs: make([]*sharedtypes.ConfigOption, 0),
									},
								},
							},
						},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - invalid RPC Type",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address:  addr1,
						Stake:    &stake1,
						Services: serviceList1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
						Services: []*sharedtypes.SupplierServiceConfig{
							{
								Service: &sharedtypes.Service{
									Id: "svcId1",
								},
								Endpoints: []*sharedtypes.SupplierEndpoint{
									{
										Url:     "http://localhost:8081",
										RpcType: sharedtypes.RPCType_UNKNOWN_RPC,
										Configs: make([]*sharedtypes.ConfigOption, 0),
									},
								},
							},
						},
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated claim",
			genState: &types.GenesisState{
				ClaimList: []types.Claim{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
