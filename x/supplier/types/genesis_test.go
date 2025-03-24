package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
	"github.com/pokt-network/pocket/x/supplier/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))
	serviceConfig1 := &sharedtypes.SupplierServiceConfig{
		ServiceId: "svcId1",
		Endpoints: []*sharedtypes.SupplierEndpoint{
			{
				Url:     "http://localhost:8081",
				RpcType: sharedtypes.RPCType_JSON_RPC,
				Configs: make([]*sharedtypes.ConfigOption, 0),
			},
		},
		RevShare: []*sharedtypes.ServiceRevenueShare{
			{
				Address:            addr1,
				RevSharePercentage: 100,
			},
		},
	}
	serviceList1 := []*sharedtypes.SupplierServiceConfig{serviceConfig1}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))
	serviceConfig2 := &sharedtypes.SupplierServiceConfig{
		ServiceId: "svcId2",
		Endpoints: []*sharedtypes.SupplierEndpoint{
			{
				Url:     "http://localhost:8082",
				RpcType: sharedtypes.RPCType_GRPC,
				Configs: make([]*sharedtypes.ConfigOption, 0),
			},
		},
		RevShare: []*sharedtypes.ServiceRevenueShare{
			{
				Address:            addr2,
				RevSharePercentage: 100,
			},
		},
	}
	serviceList2 := []*sharedtypes.SupplierServiceConfig{serviceConfig2}

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
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &stake2,
						Services:        serviceList2,
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "invalid - zero supplier stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - negative supplier stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to duplicated supplier operator address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake2,
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to nil supplier stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           nil,
						Services:        serviceList2,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to missing supplier stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
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
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &stake2,
						// Services explicitly omitted
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - empty services list",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &stake2,
						Services:        []*sharedtypes.SupplierServiceConfig{},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - invalid URL",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &stake2,
						Services: []*sharedtypes.SupplierServiceConfig{
							{
								ServiceId: "svcId1",
								Endpoints: []*sharedtypes.SupplierEndpoint{
									{
										Url:     "invalid URL",
										RpcType: sharedtypes.RPCType_JSON_RPC,
										Configs: make([]*sharedtypes.ConfigOption, 0),
									},
								},
								RevShare: []*sharedtypes.ServiceRevenueShare{
									{
										Address:            addr2,
										RevSharePercentage: 100,
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
				Params: types.DefaultParams(),
				SupplierList: []sharedtypes.Supplier{
					{
						OwnerAddress:    addr1,
						OperatorAddress: addr1,
						Stake:           &stake1,
						Services:        serviceList1,
					},
					{
						OwnerAddress:    addr2,
						OperatorAddress: addr2,
						Stake:           &stake2,
						Services: []*sharedtypes.SupplierServiceConfig{
							{
								ServiceId: "svcId1",
								Endpoints: []*sharedtypes.SupplierEndpoint{
									{
										Url:     "http://localhost:8081",
										RpcType: sharedtypes.RPCType_UNKNOWN_RPC,
										Configs: make([]*sharedtypes.ConfigOption, 0),
									},
								},
								RevShare: []*sharedtypes.ServiceRevenueShare{
									{
										Address:            addr2,
										RevSharePercentage: 100,
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
