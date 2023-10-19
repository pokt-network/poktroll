package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
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
