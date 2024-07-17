package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgAddService_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgAddService
		expectedErr error
	}{
		{
			desc: "invalid service supplier address - no service",
			msg: MsgAddService{
				Address: "invalid_address",
				// Service: intentionally omitted,
			},
			expectedErr: ErrServiceInvalidAddress,
		}, {
			desc: "valid service supplier address - no service ID",
			msg: MsgAddService{
				Address: sample.AccAddress(),
				Service: sharedtypes.Service{Name: "service name"}, // ID intentionally omitted
			},
			expectedErr: ErrServiceMissingID,
		}, {
			desc: "valid service supplier address - no service name",
			msg: MsgAddService{
				Address: sample.AccAddress(),
				Service: sharedtypes.Service{Id: "svc1"}, // Name intentionally omitted
			},
			expectedErr: ErrServiceMissingName,
		}, {
			desc: "valid service supplier address and service",
			msg: MsgAddService{
				Address: sample.AccAddress(),
				Service: sharedtypes.Service{Id: "svc1", Name: "service name"},
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
