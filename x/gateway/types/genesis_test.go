package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", math.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", math.NewInt(100))

	tests := []struct {
		desc     string
		genState *gatewaytypes.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: gatewaytypes.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &gatewaytypes.GenesisState{
				Params: gatewaytypes.DefaultParams(),
				GatewayList: []gatewaytypes.Gateway{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
					},
				},
				// this line is used by starport scaffolding # gatewaytypes/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - duplicated gateway address",
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
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
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
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
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
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
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative gateway stake",
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &gatewaytypes.GenesisState{
				GatewayList: []gatewaytypes.Gateway{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # gatewaytypes/genesis/testcase
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.genState.Validate()
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
