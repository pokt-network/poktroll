package types_test

import (
	"testing"

	"pocket/testutil/sample"
	"pocket/x/gateway/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))

	appAddr1 := sample.AccAddress()
	appAddr2 := sample.AccAddress()

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
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{appAddr1},
					},
					{
						Address:                       addr2,
						Stake:                         &stake2,
						DelegatorApplicationAddresses: []string{appAddr2},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - duplicated gateway address",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr1,
						Stake:                         &stake2,
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - nil gateway stake",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr2,
						Stake:                         nil,
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing gateway stake",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address: addr2,
						// Stake:   stake2,
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - zero gateway stake",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr2,
						Stake:                         &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative gateway stake",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr2,
						Stake:                         &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr2,
						Stake:                         &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{},
					},
					{
						Address:                       addr2,
						Stake:                         &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
						DelegatorApplicationAddresses: []string{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - invalid delegator application address",
			genState: &types.GenesisState{
				GatewayList: []types.Gateway{
					{
						Address:                       addr1,
						Stake:                         &stake1,
						DelegatorApplicationAddresses: []string{appAddr1},
					},
					{
						Address:                       addr2,
						Stake:                         &stake2,
						DelegatorApplicationAddresses: []string{"invalid app address"},
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
