package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
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
		Name: svc1.Name,
	}

	tests := []struct {
		desc        string
		genState    *types.GenesisState
		expectedErr error
	}{
		{
			desc:        "default is valid",
			genState:    types.DefaultGenesis(),
			expectedErr: nil,
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
			expectedErr: nil,
		},
		{
			desc: "invalid - duplicate service ID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ServiceList: []sharedtypes.Service{
					*svc1, *svc1,
				},
			},
			expectedErr: types.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - duplicate service name",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ServiceList: []sharedtypes.Service{
					*svc1, *svc3,
				},
			},
			expectedErr: types.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - invalid add service fee parameter (below minimum)",
			genState: &types.GenesisState{
				Params: types.Params{
					AddServiceFee: &sdk.Coin{Denom: volatile.DenomuPOKT, Amount: math.NewInt(999999)}, // 0.999999 POKT
				},
				ServiceList: []sharedtypes.Service{
					*svc1, *svc2,
				},
			},
			expectedErr: types.ErrServiceInvalidServiceFee,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.genState.Validate()
			if test.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
			}
		})
	}
}
