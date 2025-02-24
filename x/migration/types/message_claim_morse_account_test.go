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

func TestMsgClaimMorseAccount_ValidateBasic(t *testing.T) {
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
				MorseSignature:     "mock_signature",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			desc: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     "mock_signature",
			},
			err: migrationtypes.ErrMorseAccountClaim,
		}, {
			desc: "valid claim message",
			msg: migrationtypes.MsgClaimMorseAccount{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
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
	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morsePublicKey := morsePrivKey.PubKey()

	t.Run("invalid Morse signature", func(t *testing.T) {
		msg := migrationtypes.MsgClaimMorseAccount{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    sample.MorseAddressHex(),
			MorseSignature:     hex.EncodeToString([]byte("invalid_signature")),
		}

		expectedErr := migrationtypes.ErrMorseAccountClaim.Wrapf("morseSignature is invalid")
		err := msg.ValidateMorseSignature(morsePublicKey)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("valid Morse signature", func(t *testing.T) {
		msg := migrationtypes.MsgClaimMorseAccount{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    sample.MorseAddressHex(),
			// MorseSignature:  (intenionally omitted; set in #SignMorseSignature)
		}
		err := msg.SignMorseSignature(morsePrivKey)
		require.NoError(t, err)

		err = msg.ValidateMorseSignature(morsePublicKey)
		require.NoError(t, err)
	})
}
