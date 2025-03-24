package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))
	svc1AppConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))
	svc2AppConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc2"}

	emptyDelegatees := make([]string, 0)
	gatewayAddr1 := sample.AccAddress()
	gatewayAddr2 := sample.AccAddress()

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
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1, gatewayAddr2},
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr2, gatewayAddr1},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "invalid - zero app stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - negative application stake",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to duplicated app address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr1,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to nil app stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     nil,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to missing app stake",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address: addr2,
						// Stake explicitly omitted
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub key",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{"invalid address"},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub keys",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1},
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{"invalid address", gatewayAddr2},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service config not present",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						// ServiceConfigs explicitly omitted
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - empty service config",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service ID too long",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{ServiceId: "TooLongId1234567890"},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service ID with invalid characters",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{ServiceId: "12 45 !"},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - MaxDelegatedGateways less than 1",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 0,
				},
			},
			isValid: false,
		},
		{
			desc: "duplicated application",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1, gatewayAddr2},
					},
					{
						Address:                   addr1,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr2, gatewayAddr1},
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
