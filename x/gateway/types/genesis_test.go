package types_test

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))

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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr1,
						Stake:   &stake2,
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   nil,
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						// Stake:   stake2,
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
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
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
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
