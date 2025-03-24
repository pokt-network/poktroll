package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/pokt-network/pocket/x/migration/types"
)

func TestGenesisState_Validate(t *testing.T) {
	duplicateMorseAddress := sample.MorseAddressHex()

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

				MorseClaimableAccountList: []types.MorseClaimableAccount{
					{
						MorseSrcAddress: sample.MorseAddressHex(),
					},
					{
						MorseSrcAddress: sample.MorseAddressHex(),
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated morseClaimableAccount",
			genState: &types.GenesisState{
				MorseClaimableAccountList: []types.MorseClaimableAccount{
					{
						MorseSrcAddress: duplicateMorseAddress,
					},
					{
						MorseSrcAddress: duplicateMorseAddress,
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
