package types

import (
	"testing"

	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgRecoverMorseAccount_ValidateBasic(t *testing.T) {
	validMorseSrcAddress := cometcrypto.GenPrivKey().PubKey().Address().String()
	tests := []struct {
		name          string
		msg           MsgRecoverMorseAccount
		expectedError error
	}{
		{
			name: "valid message",
			msg: MsgRecoverMorseAccount{
				Authority:          sample.AccAddressBech32(),
				ShannonDestAddress: sample.AccAddressBech32(),
				MorseSrcAddress:    validMorseSrcAddress,
			},
		},
		{
			name: "invalid authority address",
			msg: MsgRecoverMorseAccount{
				Authority:          "invalid_address",
				ShannonDestAddress: sample.AccAddressBech32(),
				MorseSrcAddress:    validMorseSrcAddress,
			},
			expectedError: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid shannon destination address",
			msg: MsgRecoverMorseAccount{
				Authority:          sample.AccAddressBech32(),
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    validMorseSrcAddress,
			},
			expectedError: sdkerrors.ErrInvalidAddress,
		},
		// Do not validate MorseSrcAddress during recovery since it could be invalid
		// (e.g. too short/long, non-hex, module...)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
		})
	}
}
