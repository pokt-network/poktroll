package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
)

func TestGenesisState_Validate(t *testing.T) {
	svc1 := &shared.Service{
		Id:   "svcId1",
		Name: "svcName1",
	}

	svc2 := &shared.Service{
		Id:   "svcId2",
		Name: "svcName2",
	}

	svc3 := &shared.Service{
		Id:   "svcId3",
		Name: svc1.Name,
	}

	tests := []struct {
		desc        string
		genState    *service.GenesisState
		expectedErr error
	}{
		{
			desc:        "default is valid",
			genState:    service.DefaultGenesis(),
			expectedErr: nil,
		},
		{
			desc: "valid genesis state",
			genState: &service.GenesisState{
				Params: service.DefaultParams(),
				ServiceList: []shared.Service{
					*svc1, *svc2,
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			expectedErr: nil,
		},
		{
			desc: "invalid - duplicate service ID",
			genState: &service.GenesisState{
				Params: service.DefaultParams(),
				ServiceList: []shared.Service{
					*svc1, *svc1,
				},
			},
			expectedErr: service.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - duplicate service name",
			genState: &service.GenesisState{
				Params: service.DefaultParams(),
				ServiceList: []shared.Service{
					*svc1, *svc3,
				},
			},
			expectedErr: service.ErrServiceDuplicateIndex,
		},
		{
			desc: "invalid - invalid add service fee parameter (below minimum)",
			genState: &service.GenesisState{
				Params: service.Params{
					AddServiceFee: 999999, // 0.999999 POKT
				},
				ServiceList: []shared.Service{
					*svc1, *svc2,
				},
			},
			expectedErr: service.ErrServiceInvalidServiceFee,
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
