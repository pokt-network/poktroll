package application_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))
	svc1AppConfig := &shared.ApplicationServiceConfig{
		Service: &shared.Service{Id: "svc1"},
	}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))
	svc2AppConfig := &shared.ApplicationServiceConfig{
		Service: &shared.Service{Id: "svc2"},
	}

	emptyDelegatees := make([]string, 0)
	gatewayAddr1 := sample.AccAddress()
	gatewayAddr2 := sample.AccAddress()

	tests := []struct {
		desc     string
		genState *application.GenesisState
		isValid  bool
	}{
		{
			desc:     "default is valid",
			genState: application.DefaultGenesis(),
			isValid:  true,
		},
		{
			desc: "valid genesis state",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1, gatewayAddr2},
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr2, gatewayAddr1},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "invalid - zero app stake",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - negative application stake",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to duplicated app address",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr1,
						Stake:                     &stake2,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to nil app stake",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     nil,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to missing app stake",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address: addr2,
						// Stake explicitly omitted
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub key",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{"invalid address"},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub keys",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1},
					},
					{
						Address:                   addr2,
						Stake:                     &stake2,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
						DelegateeGatewayAddresses: []string{"invalid address", gatewayAddr2},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service config not present",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
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
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service ID too long",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*shared.ApplicationServiceConfig{
							{Service: &shared.Service{Id: "TooLongId1234567890"}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service name too long",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*shared.ApplicationServiceConfig{
							{Service: &shared.Service{
								Id:   "123",
								Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
							}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - service ID with invalid characters",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 7,
				},
				ApplicationList: []application.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*shared.ApplicationServiceConfig{
							{Service: &shared.Service{Id: "12 45 !"}},
						},
						DelegateeGatewayAddresses: emptyDelegatees,
					},
				},
			},
			isValid: false,
		},
		{
			desc: "invalid - MaxDelegatedGateways less than 1",
			genState: &application.GenesisState{
				Params: application.Params{
					MaxDelegatedGateways: 0,
				},
			},
			isValid: false,
		},
		{
			desc: "duplicated application",
			genState: &application.GenesisState{
				ApplicationList: []application.Application{
					{
						Address:                   addr1,
						Stake:                     &stake1,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc1AppConfig},
						DelegateeGatewayAddresses: []string{gatewayAddr1, gatewayAddr2},
					},
					{
						Address:                   addr1,
						Stake:                     &stake2,
						ServiceConfigs:            []*shared.ApplicationServiceConfig{svc2AppConfig},
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
