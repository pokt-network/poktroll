package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
	"pocket/x/application/types"
	sharedtypes "pocket/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))
	svc1AppConfig := &sharedtypes.ApplicationServiceConfig{
		ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
	}

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))
	svc2AppConfig := &sharedtypes.ApplicationServiceConfig{
		ServiceId: &sharedtypes.ServiceId{Id: "svc2"},
	}

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

				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          &stake2,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - zero app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative application stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to duplicated app address",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr1,
						Stake:          &stake2,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to nil app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address:        addr2,
						Stake:          nil,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to missing app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc1AppConfig},
					},
					{
						Address: addr2,
						// Explicitly missing stake
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc2AppConfig},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service config not present",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						// ServiceConfigs: omitted
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - empty service config",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:        addr1,
						Stake:          &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service ID too long",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,		
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{ServiceId: &sharedtypes.ServiceId{Id: "12345678901"}},
						},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service name too long",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{ServiceId: &sharedtypes.ServiceId{
								Id:   "123",
								Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
							}},
						},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - service ID with invalid characters",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address: addr1,
						Stake:   &stake1,
						ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
							{ServiceId: &sharedtypes.ServiceId{Id: "12 45 !"}},
						},
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
