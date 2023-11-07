package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUnstakeSupplier_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnstakeSupplier
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUnstakeSupplier{
				Address: "invalid_address",
			},
			err: ErrSupplierInvalidAddress,
		}, {
			name: "missing address",
			msg:  MsgUnstakeSupplier{},
			err:  ErrSupplierInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUnstakeSupplier{
				Address: sample.AccAddress(),
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
