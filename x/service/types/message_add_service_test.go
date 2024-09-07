package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgAddService_ValidateBasic(t *testing.T) {
	serviceOwnerAddress := sample.AccAddress()
	tests := []struct {
		desc        string
		msg         MsgAddService
		expectedErr error
	}{
		{
			desc: "invalid service owner address - no service",
			msg: MsgAddService{
				OwnerAddress: "invalid_address",
				// Service: intentionally omitted,
			},
			expectedErr: ErrServiceInvalidAddress,
		}, {
			desc: "valid service owner address - no service ID",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service:      sharedtypes.Service{Name: "service name", OwnerAddress: serviceOwnerAddress}, // ID intentionally omitted
			},
			expectedErr: ErrServiceMissingID,
		}, {
			desc: "valid service owner address - no service name",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service:      sharedtypes.Service{Id: "svc1", OwnerAddress: serviceOwnerAddress}, // Name intentionally omitted
			},
			expectedErr: ErrServiceMissingName,
		}, {
			desc: "valid service owner address - zero compute units per relay",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: 0,
					OwnerAddress:         serviceOwnerAddress,
				},
			},
			expectedErr: ErrServiceInvalidComputeUnitsPerRelay,
		}, {
			desc: "signer address does not equal service owner address",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         sample.AccAddress(), // Random address that does not equal serviceOwnerAddress
				},
			},
			expectedErr: ErrServiceInvalidOwnerAddress,
		},
		{
			desc: "valid msg add service",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         serviceOwnerAddress,
				},
			},
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateComputeUnitsPerRelay(t *testing.T) {
	tests := []struct {
		desc                 string
		computeUnitsPerRelay uint64
		expectedErr          error
	}{
		{
			desc:                 "zero compute units per relay",
			computeUnitsPerRelay: 0,
			expectedErr:          ErrServiceInvalidComputeUnitsPerRelay,
		}, {
			desc:                 "valid compute units per relay",
			computeUnitsPerRelay: 1,
			expectedErr:          nil,
		}, {
			desc:                 "max compute units per relay",
			computeUnitsPerRelay: ComputeUnitsPerRelayMax,
			expectedErr:          nil,
		}, {
			desc:                 "compute units per relay greater than max",
			computeUnitsPerRelay: ComputeUnitsPerRelayMax + 1,
			expectedErr:          ErrServiceInvalidComputeUnitsPerRelay,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := ValidateComputeUnitsPerRelay(test.computeUnitsPerRelay)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
