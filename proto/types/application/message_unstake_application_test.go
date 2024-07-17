package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUnstakeApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUnstakeApplication
		expectedErr error
	}{
		{
			desc: "valid",
			msg: MsgUnstakeApplication{
				Address: sample.AccAddress(),
			},
		},
		{
			desc:        "invalid - missing address",
			msg:         MsgUnstakeApplication{},
			expectedErr: ErrAppInvalidAddress,
		},
		{
			desc: "invalid - invalid address",
			msg: MsgUnstakeApplication{
				Address: "invalid_address",
			},
			expectedErr: ErrAppInvalidAddress,
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
