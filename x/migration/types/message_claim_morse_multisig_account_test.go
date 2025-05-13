package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	cometed "github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func TestMsgClaimMorseMultiSigAccount_ValidateBasic(t *testing.T) {

	morsePrivKeys := testmigration.GenMorsePrivateKeysForMultiSig(0)

	wrongMorsePrivKeys := testmigration.GenMorsePrivateKeysForMultiSig(99)

	t.Run("invalid Shannon destination address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseMultiSigAccount(
			"invalid_shannon_address",
			morsePrivKeys,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.ErrorContains(t, err, fmt.Sprintf("invalid shannonDestAddress address (%s)", msg.GetShannonDestAddress()))
	})

	t.Run("invalid Morse signature", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseMultiSigAccount(
			sample.AccAddress(),
			morsePrivKeys,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		// Set the Morse signature to a non-hex string to simulate a corrupt signature.
		msg.MorseSignature = []byte("invalid_signature")

		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"signature verification failed",
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("wrong Morse signature", func(t *testing.T) {
		// Construct a valid MsgClaimMorseMultiSigAccount message using the "wrong" Morse
		// private key. This populates the signature with a valid signature, but
		// corresponding to the wrong key and address.
		msg, err := migrationtypes.NewMsgClaimMorseMultiSigAccount(
			sample.AccAddress(),
			wrongMorsePrivKeys,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		var pubKeys []cometed.PubKey
		for _, priv := range morsePrivKeys {
			edPub, _ := priv.PubKey().(cometed.PubKey)
			pubKeys = append(pubKeys, edPub)
		}
		// Reset the morsePublicKey fields, leaving the "wrong" signature in place.
		msg.MorsePublicKeys = pubKeys
		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"signature verification failed",
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("valid Morse claim account message", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseMultiSigAccount(
			sample.AccAddress(),
			morsePrivKeys,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.NoError(t, err)
	})
}
