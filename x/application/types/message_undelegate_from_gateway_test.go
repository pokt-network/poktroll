package types

import (
	"testing"

	"pocket/testutil/sample"

	"github.com/stretchr/testify/require"
)

func TestMsgUndelegateFromGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUndelegateFromGateway
		err  error
	}{
		{
			name: "invalid app address - no gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress: "invalid_address",
				// GatewayAddress: sample.AccAddress(),
			},
			err: ErrAppInvalidAddress,
		}, {
			name: "valid app address - no gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress: sample.AccAddress(),
				// GatewayAddress: sample.AccAddress(),
			},
			err: ErrAppInvalidGatewayAddress,
		}, {
			name: "valid app address - invalid gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress:     sample.AccAddress(),
				GatewayAddress: "invalid_address",
			},
			err: ErrAppInvalidGatewayAddress,
		}, {
			name: "valid address",
			msg: MsgUndelegateFromGateway{
				AppAddress:     sample.AccAddress(),
				GatewayAddress: sample.AccAddress(),
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
