package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	ownerAddr := sample.AccAddressBech32()
	svc1 := &sharedtypes.Service{
		Id:                   "svcId1",
		Name:                 "svcName1",
		OwnerAddress:         ownerAddr,
		ComputeUnitsPerRelay: 1,
	}

	svc2 := &sharedtypes.Service{
		Id:                   "svcId2",
		Name:                 "svcName2",
		OwnerAddress:         ownerAddr,
		ComputeUnitsPerRelay: 1,
	}

	svc3 := &sharedtypes.Service{
		Id:                   "svcId3",
		Name:                 svc1.Name,
		OwnerAddress:         ownerAddr,
		ComputeUnitsPerRelay: 1,
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
			expectedErr: nil,
		},
		{
			desc: "invalid - duplicated relayMiningDifficulty",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				RelayMiningDifficultyList: []types.RelayMiningDifficulty{
					{
						ServiceId: "0",
					},
					{
						ServiceId: "0",
					},
				},
			},
			expectedErr: types.ErrServiceDuplicateIndex,
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
					AddServiceFee: &sdk.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(999999)}, // 0.999999 POKT
				},
				ServiceList: []sharedtypes.Service{
					*svc1, *svc2,
				},
			},
			expectedErr: types.ErrServiceParamInvalid,
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
