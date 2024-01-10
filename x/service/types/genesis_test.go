package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	srv1 := &sharedtypes.Service{
		Id:   "srv1",
		Name: "srv1",
	}

	srv2 := &sharedtypes.Service{
		Id:   "srv2",
		Name: "srv2",
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
				ServiceList: []sharedtypes.Service{
					*srv1, *srv2,
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - duplicate service ID",
			genState: &types.GenesisState{
				ServiceList: []sharedtypes.Service{
					*srv1, *srv1,
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
