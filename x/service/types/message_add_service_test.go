package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgAddService_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgAddService
		expectedErr error
	}{
		{
			desc: "invalid supplier address - no service",
			msg: MsgAddService{
				SupplierAddress: "invalid_address",
				// Service: intentionally omitted,
			},
			expectedErr: ErrServiceInvalidAddress,
		}, {
			desc: "valid supplier address - no service ID",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Name: "service name"}, // ID intentionally omitted
			},
			expectedErr: ErrServiceMissingID,
		}, {
			desc: "valid supplier address - no service name",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Id: "srv1"}, // Name intentionally omitted
			},
			expectedErr: ErrServiceMissingName,
		}, {
			desc: "valid supplier address and service",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Id: "srv1", Name: "service name"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
