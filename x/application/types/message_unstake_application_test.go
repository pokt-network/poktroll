package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUnstakeApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnstakeApplication
		err  error
	}{
		{
			name: "valid",
			msg: MsgUnstakeApplication{
				Address: sample.AccAddress(),
			},
		},
		{
			name: "invalid - missing address",
			msg:  MsgUnstakeApplication{},
			err:  ErrAppInvalidAddress,
		},
		{
			name: "invalid - invalid address",
			msg: MsgUnstakeApplication{
				Address: "invalid_address",
			},
			err: ErrAppInvalidAddress,
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
