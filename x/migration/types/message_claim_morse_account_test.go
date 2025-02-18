package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgClaimMorseAccount_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgClaimMorseAccount
		err  error
	}{
		{
			name: "invalid ShannonDestAddress",
			msg: MsgClaimMorseAccount{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     "mock_signature",
			},
			err: ErrMorseAccountClaim,
		}, {
			name: "valid claim message",
			msg: MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
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
