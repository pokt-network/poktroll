package types

import (
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/encoding"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	_ sdk.Msg           = (*MsgClaimMorseSupplier)(nil)
	_ morseClaimMessage = (*MsgClaimMorseSupplier)(nil)
)

// NewMsgClaimMorseSupplier creates a new MsgClaimMorseSupplier.
// If morsePrivateKey is provided (i.e. not nil), it is used to sign the message.
// morsePrivateKey MUST be ONE OF THE FOLLOWING:
//   - The Morse node private key (i.e. operator); a.k.a custodial
//   - The Morse output private key (i.e. owner); a.k.a non-custodial
func NewMsgClaimMorseSupplier(
	shannonOwnerAddress string,
	shannonOperatorAddress string,
	morseNodeAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	services []*sharedtypes.SupplierServiceConfig,
	shannonSigningAddr string,
) (*MsgClaimMorseSupplier, error) {
	morseNodeAddress = encoding.NormalizeMorseAddress(morseNodeAddress)

	msg := &MsgClaimMorseSupplier{
		MorseNodeAddress:       morseNodeAddress,
		ShannonOwnerAddress:    shannonOwnerAddress,
		ShannonOperatorAddress: shannonOperatorAddress,
		Services:               services,
		ShannonSigningAddress:  shannonSigningAddr,
	}

	if morsePrivateKey != nil {
		msg.MorsePublicKey = morsePrivateKey.PubKey().Bytes()

		// Assume that the morsePrivateKey corresponds to ONE OF THE FOLLOWING:
		// - The morse node address (i.e. operator): leave signer_is_output_address as false
		// - The morse output address (i.e. owner): set signer_is_output_address to true
		// If any other private key is used, the claim message will error.
		if msg.GetMorseSignerAddress() != morseNodeAddress {
			msg.SignerIsOutputAddress = true
		}

		if err := msg.SignMorseSignature(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ValidateBasic ensures that:
// - The shannon owner address is valid (i.e. it is a valid bech32 address).
// - The shannon operator address is valid (i.e. it is a valid bech32 address).
// - The supplier service configs are valid.
// - The morsePublicKey is valid.
// - The morseSrcAddress matches the public key.
// - The morseSignature is valid.
func (msg *MsgClaimMorseSupplier) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetShannonOwnerAddress()); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid shannon owner address address (%s): %s",
			msg.GetShannonOwnerAddress(), err,
		)
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonOperatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid shannon operator address address (%s): %s",
			msg.GetShannonOperatorAddress(), err,
		)
	}

	if err := sharedtypes.ValidateSupplierServiceConfigs(msg.Services); err != nil {
		return ErrMorseSupplierClaim.Wrapf("invalid service configs: %s", err)
	}

	// Validate the Morse signature.
	if err := msg.ValidateMorseSignature(); err != nil {
		return err
	}

	return nil
}

// SignMorseSignature signs the given MsgClaimMorseApplication with the given Morse private key.
func (msg *MsgClaimMorseSupplier) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return ErrMorseSignature.Wrapf("unable to get signing bytes: %s", err)
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	if err != nil {
		return ErrMorseSignature.Wrapf("unable to sign message: %s", err)
	}

	return nil
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseSupplier
// matches the given Morse public key.
func (msg *MsgClaimMorseSupplier) ValidateMorseSignature() error {
	return validateMorseSignature(msg)
}

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseSupplier
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseSupplier) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}

// GetMorseSignerAddress returns the address associated with the Morse keypair which
// was used to sign this claim message.
// This address is expected to be ONE OF THE FOLLOWING:
// - The Morse node address (i.e. operator)
// - The Morse output address (i.e. owner)
func (msg *MsgClaimMorseSupplier) GetMorseSignerAddress() string {
	return encoding.NormalizeMorseAddress(msg.GetMorsePublicKey().Address().String())
}
