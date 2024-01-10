package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgAddService_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgAddService
		err  error
	}{
		{
			name: "invalid supplier address - no service",
			msg: MsgAddService{
				SupplierAddress: "invalid_address",
				// Service: intentionally omitted,
			},
			err: ErrServiceInvalidAddress,
		}, {
			name: "valid supplier address - no service ID",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Name: "service name"},
			},
			err: ErrServiceMissingID,
		}, {
			name: "valid supplier address - no service name",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Id: "srv1"},
			},
			err: ErrServiceMissingName,
		}, {
			name: "valid address and service",
			msg: MsgAddService{
				SupplierAddress: sample.AccAddress(),
				Service:         sharedtypes.Service{Id: "srv1", Name: "service name"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
