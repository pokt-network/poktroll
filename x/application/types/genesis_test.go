package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))
	svc1AppConfig := &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc1"},
	}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))
	svc2AppConfig := &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc2"},
	}

	emptyDelegatees := make([]string, 0)
	gatewayAddr1 := sample.AccAddress()
	gatewayAddr2 := sample.AccAddress()

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
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
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
			valid: true,
		},
		{
			desc: "invalid - zero app stake",
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
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
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
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
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
						Stake:                     &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
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
						Stake:                     &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to duplicated app address",
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
						Address:                   addr1,
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to nil app stake",
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
						Stake:                     nil,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to missing app stake",
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
						Address: addr2,
						// Explicitly missing stake
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub key",
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
						Stake:                     &stake2,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{"invalid address"},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub keys",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
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
			valid: false,
		},
		{
			desc: "invalid - service config not present",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						// ServiceConfigs: omitted
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - empty service config",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service ID too long",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{Service: &sharedtypes.Service{Id: "12345678901"}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service name too long",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{Service: &sharedtypes.Service{
								Id:   "123",
								Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
							}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service ID with invalid characters",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{Service: &sharedtypes.Service{Id: "12 45 !"}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - MaxDelegatedGateways less than 1",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxDelegatedGateways: 0,
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
