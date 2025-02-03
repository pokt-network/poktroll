package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgCreateMorseAccountState_ValidateBasic(t *testing.T) {
	validMsg, err := NewMsgCreateMorseAccountState(sample.AccAddress(), MorseAccountState{})
	require.NoError(t, err)

	tests := []struct {
		name string
		msg  MsgCreateMorseAccountState
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateMorseAccountState{
				Authority: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg:  *validMsg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
