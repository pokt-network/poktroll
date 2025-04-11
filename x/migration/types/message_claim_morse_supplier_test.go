package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	testEndpoints = []*sharedtypes.SupplierEndpoint{
		{
			Url:     "http://test.example:1234",
			RpcType: sharedtypes.RPCType_JSON_RPC,
		},
	}

	testRevShare = []*sharedtypes.ServiceRevenueShare{
		{
			Address:            sample.AccAddress(),
			RevSharePercentage: uint64(100),
		},
	}

	testSupplierServiceConfigs = []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: testServiceId,
			Endpoints: testEndpoints,
			RevShare:  testRevShare,
		},
	}
)

func TestMsgClaimMorseSupplier_ValidateBasic(t *testing.T) {
	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	wrongMorsePrivKey := testmigration.GenMorsePrivateKey(99)

	t.Run("invalid Shannon owner address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			"invalid_address",
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.ErrorContains(t, err, fmt.Sprintf("invalid shannon owner address address (%s)", msg.GetShannonOwnerAddress()))
	})

	t.Run("invalid Shannon operator address", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			"invalid_address",
			morsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.ErrorContains(t, err, fmt.Sprintf("invalid shannon operator address address (%s)", msg.GetShannonOperatorAddress()))
	})

	t.Run("invalid Morse signature", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServiceConfigs,
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
		// Construct a valid MsgClaimMorseSupplier message using the "wrong"
		// Morse private key. This populates the signature with a valid signature,
		// but corresponding to the wrong key and address.
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			wrongMorsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
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
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
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
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
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

	t.Run("invalid service ID", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			[]*sharedtypes.SupplierServiceConfig{
				{ServiceId: "invalid_service_id"},
			},
			sample.AccAddress(),
		)
		require.NoError(t, err)

		// Set the morseSrcAddress to the wrong address.
		msg.MorseSrcAddress = wrongMorsePrivKey.PubKey().Address().String()

		expectedErr := migrationtypes.ErrMorseSupplierClaim.Wrapf(
			"invalid service configs: %s",
			sharedtypes.ErrSharedInvalidService.Wrapf(
				"invalid service ID: %s",
				msg.GetServices()[0].GetServiceId(),
			),
		)

		err = msg.ValidateBasic()
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("valid Morse claim account message", func(t *testing.T) {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServiceConfigs,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		err = msg.ValidateBasic()
		require.NoError(t, err)
	})
}
