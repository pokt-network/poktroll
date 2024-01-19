package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	svc1 := &sharedtypes.Service{
		Id:   "svcId1",
		Name: "svcName1",
	}

	svc2 := &sharedtypes.Service{
		Id:   "svcId2",
		Name: "svcName2",
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
					*svc1, *svc2,
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - duplicate service ID",
			genState: &types.GenesisState{
				ServiceList: []sharedtypes.Service{
					*svc1, *svc1,
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
