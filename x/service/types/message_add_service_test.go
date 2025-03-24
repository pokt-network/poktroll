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
		},
		{
			desc: "no service ID",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					// ID intentionally omitted
					Name:                 "service name",
					OwnerAddress:         serviceOwnerAddress,
					ComputeUnitsPerRelay: 1,
				},
			},
			expectedErr: sharedtypes.ErrSharedInvalidService.Wrapf("invalid service ID: %q", ""),
		},
		{
			desc: "no service name",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id: "svc1",
					// Name intentionally omitted
					OwnerAddress:         serviceOwnerAddress,
					ComputeUnitsPerRelay: 1,
				},
			},
			expectedErr: nil,
		},
		{
			desc: "invalid service name",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service&name",
					OwnerAddress:         serviceOwnerAddress,
					ComputeUnitsPerRelay: 1,
				},
			},
			expectedErr: sharedtypes.ErrSharedInvalidService.Wrapf("invalid service name: %q", "service&name"),
		},
		{
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
			desc: "zero compute units per relay",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: 0,
					OwnerAddress:         serviceOwnerAddress,
				},
			},
			expectedErr: sharedtypes.ErrSharedInvalidService.
				Wrapf("%s", sharedtypes.ErrSharedInvalidComputeUnitsPerRelay),
		},
		{
			desc: "compute units per relay greater than max",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: sharedtypes.ComputeUnitsPerRelayMax + 1,
					OwnerAddress:         serviceOwnerAddress,
				},
			},
			expectedErr: sharedtypes.ErrSharedInvalidService.
				Wrapf("%s", sharedtypes.ErrSharedInvalidComputeUnitsPerRelay),
		},
		{
			desc: "min compute units per relay",
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
		{
			desc: "max compute units per relay",
			msg: MsgAddService{
				OwnerAddress: serviceOwnerAddress,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "service name",
					ComputeUnitsPerRelay: sharedtypes.ComputeUnitsPerRelayMax,
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
				require.ErrorContains(t, err, test.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
