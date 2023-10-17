package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
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
			err: ErrSample,
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
