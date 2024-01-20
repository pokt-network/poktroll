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

	svc3 := &sharedtypes.Service{
		Id:   "svcId3",
		Name: "svcName1",
	}

	tests := []struct {
		desc          string
		genState      *types.GenesisState
		expectedError error
	}{
		{
			desc:          "default is valid",
			genState:      types.DefaultGenesis(),
			expectedError: nil,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ServiceList: []sharedtypes.Service{
					*svc1, *svc2,
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			expectedError: nil,
		},
		{
			desc: "invalid - duplicate service ID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ServiceList: []sharedtypes.Service{
					*svc1, *svc1,
				},
			},
			expectedError: types.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - duplicate service name",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ServiceList: []sharedtypes.Service{
					*svc1, *svc3,
				},
			},
			expectedError: types.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - invalid add service fee parameter",
			genState: &types.GenesisState{
				Params: types.Params{
					AddServiceFee: 999999, // 0.999999 POKT
				},
				ServiceList: []sharedtypes.Service{
					*svc1, *svc2,
				},
			},
			expectedError: types.ErrServiceInvalidServiceFee,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expectedError)
			}
		})
	}
}
