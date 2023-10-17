package types

import (
	"testing"

	"pocket/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
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
			err:  sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid - invalid address",
			msg: MsgUnstakeApplication{
				Address: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
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
