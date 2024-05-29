package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGenesisState_Validate(t *testing.T) {
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
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
				RelayMiningDifficultyList: []types.RelayMiningDifficulty{
					{
						ServiceId: "0",
					},
					{
						ServiceId: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "invalid genesis state - ComputeUnitsToTokensMultiplier is 0",
			genState: &types.GenesisState{
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 0,
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: false,
		},
		{
			desc: "duplicated relayMiningDifficulty",
			genState: &types.GenesisState{
				RelayMiningDifficultyList: []types.RelayMiningDifficulty{
					{
						ServiceId: "0",
					},
					{
						ServiceId: "0",
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
