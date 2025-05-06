package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgRecoverMorseAccount_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRecoverMorseAccount
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgRecoverMorseAccount{
				Authority: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgRecoverMorseAccount{
				Authority: sample.AccAddress(),
			},
		},

		// TODO_MAINNET_MIGRATION(@bryanchriswhite): Add coverage for the following cases:
		// - MorseSrcAddress is not a valid address
		// - ShannonDestAddress is not a valid address
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
