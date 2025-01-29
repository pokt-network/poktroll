package types

import (
	"encoding/hex"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgClaimMorsePokt_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgClaimMorsePokt
		err  error
	}{
		{
			name: "invalid source address",
			msg: MsgClaimMorsePokt{
				MorseSrcAddress:    "invalid_address",
				ShannonDestAddress: sample.AccAddress(),
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid address",
			msg: MsgClaimMorsePokt{
				MorseSrcAddress:    hex.EncodeToString(sample.ConsAddress().Bytes()),
				ShannonDestAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid addresses",
			msg: MsgClaimMorsePokt{
				MorseSrcAddress:    hex.EncodeToString(sample.ConsAddress().Bytes()),
				ShannonDestAddress: sample.AccAddress(),
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
