package types_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddressBech32()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))
	svc1AppConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}

	addr2 := sample.AccAddressBech32()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))
	svc2AppConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc2"}

	emptyDelegatees := make([]string, 0)
	gatewayAddr1 := sample.AccAddressBech32()
	gatewayAddr2 := sample.AccAddressBech32()

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
							{ServiceId: strings.Repeat("a", 43)}, // 42 is the max length hardcoded in the services module
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

func TestGenesisState_Validate_PerSessionSpendLimitHistory(t *testing.T) {
	stake := sdk.NewCoin("upokt", math.NewInt(100))
	valid := sdk.NewCoin("upokt", math.NewInt(2_000_000))
	tooLow := sdk.NewCoin("upokt", math.NewInt(1))
	nilAmount := sdk.Coin{Denom: "upokt"}
	genesisWith := func(
		limit *sdk.Coin,
		history ...*types.ApplicationPerSessionSpendLimitUpdate,
	) *types.GenesisState {
		return &types.GenesisState{
			Params: types.DefaultParams(),
			ApplicationList: []types.Application{{
				Address:                     sample.AccAddressBech32(),
				Stake:                       &stake,
				ServiceConfigs:              []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
				DelegateeGatewayAddresses:   []string{},
				PerSessionSpendLimit:        limit,
				PerSessionSpendLimitHistory: history,
			}},
		}
	}

	tests := []struct {
		desc     string
		genState *types.GenesisState
		isValid  bool
	}{
		{
			desc: "valid - strictly increasing history",
			genState: genesisWith(&valid,
				&types.ApplicationPerSessionSpendLimitUpdate{ActivationHeight: 1},
				&types.ApplicationPerSessionSpendLimitUpdate{PerSessionSpendLimit: &valid, ActivationHeight: 11},
			),
			isValid: true,
		},
		{
			desc: "invalid - unordered activation heights",
			genState: genesisWith(nil,
				&types.ApplicationPerSessionSpendLimitUpdate{PerSessionSpendLimit: &valid, ActivationHeight: 11},
				&types.ApplicationPerSessionSpendLimitUpdate{ActivationHeight: 5},
			),
			isValid: false,
		},
		{
			desc: "invalid - duplicate activation heights",
			genState: genesisWith(&valid,
				&types.ApplicationPerSessionSpendLimitUpdate{ActivationHeight: 11},
				&types.ApplicationPerSessionSpendLimitUpdate{PerSessionSpendLimit: &valid, ActivationHeight: 11},
			),
			isValid: false,
		},
		{
			desc: "invalid - zero activation height",
			genState: genesisWith(nil,
				&types.ApplicationPerSessionSpendLimitUpdate{ActivationHeight: 0},
			),
			isValid: false,
		},
		{
			desc:     "invalid - nil history entry",
			genState: genesisWith(nil, nil),
			isValid:  false,
		},
		{
			desc: "invalid - history limit below minimum",
			genState: genesisWith(nil,
				&types.ApplicationPerSessionSpendLimitUpdate{PerSessionSpendLimit: &tooLow, ActivationHeight: 1},
			),
			isValid: false,
		},
		{
			desc: "invalid - history limit with nil amount",
			genState: genesisWith(nil,
				&types.ApplicationPerSessionSpendLimitUpdate{PerSessionSpendLimit: &nilAmount, ActivationHeight: 1},
			),
			isValid: false,
		},
		{
			desc:     "invalid - live limit with nil amount",
			genState: genesisWith(&nilAmount),
			isValid:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.genState.Validate()
			if test.isValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, types.ErrAppInvalidPerSessionSpendLimit)
			}
		})
	}
}
