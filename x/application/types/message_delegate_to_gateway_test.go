package types

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/sample"

	"github.com/stretchr/testify/require"
)

func TestMsgDelegateToGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDelegateToGateway
		err  error
	}{
		{
			name: "invalid app address - no gateway address",
			msg: MsgDelegateToGateway{
				AppAddress: "invalid_address",
				// GatewayAddress: intentionally omitted,
			},
			err: ErrAppInvalidAddress,
		}, {
			name: "valid app address - no gateway address",
			msg: MsgDelegateToGateway{
				AppAddress: sample.AccAddress(),
				// GatewayAddress: intentionally omitted,
			},
			err: ErrAppInvalidGatewayAddress,
		}, {
			name: "valid app address - invalid gateway address",
			msg: MsgDelegateToGateway{
				AppAddress:     sample.AccAddress(),
				GatewayAddress: "invalid_address",
			},
			err: ErrAppInvalidGatewayAddress,
		}, {
			name: "valid address",
			msg: MsgDelegateToGateway{
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
