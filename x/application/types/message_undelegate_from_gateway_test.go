package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
