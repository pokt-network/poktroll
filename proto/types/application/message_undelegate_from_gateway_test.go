package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUndelegateFromGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUndelegateFromGateway
		expectedErr error
	}{
		{
			desc: "invalid app address - no gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress: "invalid_address",
				// GatewayAddress explicitly omitted
			},
			expectedErr: ErrAppInvalidAddress,
		}, {
			desc: "valid app address - no gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress: sample.AccAddress(),
				// GatewayAddress explicitly omitted
			},
			expectedErr: ErrAppInvalidGatewayAddress,
		}, {
			desc: "valid app address - invalid gateway address",
			msg: MsgUndelegateFromGateway{
				AppAddress:     sample.AccAddress(),
				GatewayAddress: "invalid_address",
			},
			expectedErr: ErrAppInvalidGatewayAddress,
		}, {
			desc: "valid address",
			msg: MsgUndelegateFromGateway{
				AppAddress:     sample.AccAddress(),
				GatewayAddress: sample.AccAddress(),
			},
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
