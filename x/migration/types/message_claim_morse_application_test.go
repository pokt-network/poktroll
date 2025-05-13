package types_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testServiceId = "svc1"

var testAppServiceConfig = sharedtypes.ApplicationServiceConfig{ServiceId: testServiceId}

func TestMsgClaimMorseApplication_ValidateBasic(t *testing.T) {
	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	wrongMorsePrivKey := testmigration.GenMorsePrivateKey(99)

	t.Run("invalid Shannon destination address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			"invalid_address",
			morsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.ErrorContains(t, err, fmt.Sprintf("invalid shannonDestAddress address (%s)", msg.GetShannonDestAddress()))
	})

	t.Run("invalid Morse signature", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		// Set the Morse signature to a non-hex string to simulate a corrupt signature.
		msg.MorseSignature = []byte("invalid_signature")

		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"invalid morse signature length; expected %d, got %d",
			migrationtypes.MorseSignatureLengthBytes,
			len(msg.GetMorseSignature()),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("wrong Morse signature", func(t *testing.T) {
		// Construct a valid MsgClaimMorseApplication message using the "wrong"
		// Morse private key. This populates the signature with a valid signature,
		// but corresponding to the wrong key and address.
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			wrongMorsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		// Reset the morsePublicKey fields, leaving the "wrong" signature in place.
		msg.MorsePublicKey = morsePrivKey.PubKey().Bytes()
		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"morseSignature (%x) is invalid for Morse address (%s)",
			msg.GetMorseSignature(),
			msg.GetMorseSrcAddress(),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("invalid service ID", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&sharedtypes.ApplicationServiceConfig{ServiceId: strings.Repeat("a", 43)},
			sample.AccAddress(),
		)
		require.NoError(t, err)

		expectedErr := sharedtypes.ErrSharedInvalidServiceId
		err = msg.ValidateBasic()
		require.ErrorContains(t, err, expectedErr.Error())
	})

	t.Run("valid Morse claim account message", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.NoError(t, err)
	})
}
