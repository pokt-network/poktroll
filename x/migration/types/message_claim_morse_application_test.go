package types_test

import (
	"fmt"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testServiceId = "svc1"

var testAppServiceConfig = sharedtypes.ApplicationServiceConfig{ServiceId: testServiceId}

func TestMsgClaimMorseApplication_ValidateBasic_XXX(t *testing.T) {
	t.SkipNow()

	tests := []struct {
		name string
		msg  migrationtypes.MsgClaimMorseApplication
		err  error
	}{
		{
			name: "invalid ShannonDestAddress",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (empty)",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "",
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (too long)",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "xxxxxxxxxxxxxxxxxxxx",
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid empty MorseSignature",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     nil,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "valid claim message",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
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

func TestMsgClaimMorseApplication_ValidateBasic(t *testing.T) {
	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	wrongMorsePrivKey := testmigration.GenMorsePrivateKey(99)

	t.Run("invalid Shannon destination address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			"invalid_address",
			morsePrivKey,
			&testAppServiceConfig,
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
		)
		require.NoError(t, err)

		// Set the Morse signature to a non-hex string to simulate a corrupt signature.
		msg.MorseSignature = []byte("invalid_signature")

		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"morseSignature (%x) is invalid for Morse address (%s)",
			msg.GetMorseSignature(),
			msg.GetMorseSrcAddress(),
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
			morsePrivKey,
			&testAppServiceConfig,
		)
		require.NoError(t, err)

		// Reset the morseSrcAddress and morsePublicKey fields, leaving
		// the "wrong" signature in place. The address MUST match the
		// key to pass validation such that this case can be covered.
		msg.MorseSrcAddress = morsePrivKey.PubKey().Address().String()
		msg.MorsePublicKey = morsePrivKey.PubKey().Bytes()
		expectedErr := migrationtypes.ErrMorseSignature.Wrapf(
			"morseSignature (%x) is invalid for Morse address (%s)",
			msg.GetMorseSignature(),
			msg.GetMorseSrcAddress(),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("invalid Morse address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&testAppServiceConfig,
		)
		require.NoError(t, err)

		// Set the morseSrcAddress to an invalid (non-bech32) address.
		msg.MorseSrcAddress = "invalid_address"

		expectedErr := migrationtypes.ErrMorseSrcAddress.Wrapf(
			"morseSrcAddress (%s) does not match morsePublicKey address (%s)",
			msg.GetMorseSrcAddress(),
			morsePrivKey.PubKey().Address().String(),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("wrong Morse address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&testAppServiceConfig,
		)
		require.NoError(t, err)

		// Set the morseSrcAddress to the wrong address.
		msg.MorseSrcAddress = wrongMorsePrivKey.PubKey().Address().String()

		expectedErr := migrationtypes.ErrMorseSrcAddress.Wrapf(
			"morseSrcAddress (%s) does not match morsePublicKey address (%s)",
			msg.GetMorseSrcAddress(),
			morsePrivKey.PubKey().Address().String(),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("valid Morse claim account message", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morsePrivKey,
			&testAppServiceConfig,
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.NoError(t, err)
	})
}
