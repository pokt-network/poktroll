package types

import (
	"testing"

	"pocket/testutil/sample"

	"github.com/stretchr/testify/require"
)

func TestMsgUnstakeGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnstakeGateway
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUnstakeGateway{
				Address: "invalid_address",
			},
			err: ErrGatewayInvalidAddress,
		}, {
			name: "missing address",
			msg:  MsgUnstakeGateway{},
			err:  ErrGatewayInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUnstakeGateway{
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
