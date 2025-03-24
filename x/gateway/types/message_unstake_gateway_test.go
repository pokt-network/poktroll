package types

import (
	"testing"

	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgUnstakeGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUnstakeGateway
		expectedErr error
	}{
		{
			desc: "invalid address",
			msg: MsgUnstakeGateway{
				Address: "invalid_address",
			},
			expectedErr: ErrGatewayInvalidAddress,
		},
		{
			desc:        "missing address",
			msg:         MsgUnstakeGateway{},
			expectedErr: ErrGatewayInvalidAddress,
		},
		{
			desc: "valid address",
			msg: MsgUnstakeGateway{
				Address: sample.AccAddress(),
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
