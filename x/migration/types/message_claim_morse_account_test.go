package types_test

import (
	"encoding/hex"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const mockMorseSignatureHex = "6c0d3b25a3e53eb6739f00ac66fc70168dfbb6dfe306a50f48a5f9d732b23068be3840a7127e1d849b4b2c54f5d34c2db36c2d6da46263cc72270f8f5dfdec5f"

var mockMorseSignature []byte

func init() {
	var err error
	mockMorseSignature, err = hex.DecodeString(mockMorseSignatureHex)
	if err != nil {
		panic(err)
	}
}

func TestMsgClaimMorseAccount_ValidateBasic(t *testing.T) {
	require.Len(t, mockMorseSignature, migrationtypes.MorseSignatureLengthBytes)

	tests := []struct {
		desc string
		msg  migrationtypes.MsgClaimMorseAccount
		err  error
	}{
		{
			desc: "invalid ShannonDestAddress",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			desc: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     mockMorseSignature,
			},
			err: migrationtypes.ErrMorseAccountClaim,
		}, {
			desc: "invalid empty MorseSignature",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     nil,
			},
			err: migrationtypes.ErrMorseAccountClaim,
		}, {
			desc: "valid claim message",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgClaimMorseAccount_ValidateMorseSignature(t *testing.T) {
	morsePrivKey := testmigration.GenMorsePrivateKey(t, 0)
	morsePublicKey := morsePrivKey.PubKey()

	t.Run("invalid Morse signature", func(t *testing.T) {
		msg := migrationtypes.MsgClaimMorseAccount{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    sample.MorseAddressHex(),
			MorseSignature:     []byte("invalid_signature"),
		}

		expectedErr := migrationtypes.ErrMorseAccountClaim.Wrapf("morseSignature is invalid")
		err := msg.ValidateMorseSignature(morsePublicKey)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("valid Morse signature", func(t *testing.T) {
		msg := migrationtypes.MsgClaimMorseAccount{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    sample.MorseAddressHex(),
			// MorseSignature:  (intenionally omitted; set in #SignMsgClaimMorseAccount)
		}
		err := msg.SignMsgClaimMorseAccount(morsePrivKey)
		require.NoError(t, err)

		err = msg.ValidateMorseSignature(morsePublicKey)
		require.NoError(t, err)
	})
}
