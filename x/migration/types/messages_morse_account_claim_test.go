package types

import (
	"encoding/hex"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgCreateMorseAccountClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateMorseAccountClaim
		err  error
	}{
		{
			name: "invalid source address",
			msg: MsgCreateMorseAccountClaim{
				MorseSrcAddress:    "invalid_address",
				ShannonDestAddress: sample.AccAddress(),
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid address",
			msg: MsgCreateMorseAccountClaim{
				MorseSrcAddress:    hex.EncodeToString(sample.ConsAddress().Bytes()),
				ShannonDestAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses",
			msg: MsgCreateMorseAccountClaim{
				MorseSrcAddress:    hex.EncodeToString(sample.ConsAddress().Bytes()),
				ShannonDestAddress: sample.AccAddress(),
			},
		}}
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
