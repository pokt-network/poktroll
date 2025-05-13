package types

import (
	cometcrypto "github.com/cometbft/cometbft/crypto"
	cometed "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
	morsecrypto "github.com/pokt-network/poktroll/x/migration/types/morsecrypto"
)

var (
	_ sdk.Msg           = (*MsgClaimMorseMultiSigAccount)(nil)
	_ MorseClaimMessage = (*MsgClaimMorseMultiSigAccount)(nil)
)

// NewMsgClaimMorseMultiSigAccount constructs a new MsgClaimMorseMultiSigAccount and signs it if privKeys are provided.
func NewMsgClaimMorseMultiSigAccount(
	shannonDestAddr string,
	morsePrivKeys []cometcrypto.PrivKey,
	shannonSignerAddr string,
) (*MsgClaimMorseMultiSigAccount, error) {

	if len(morsePrivKeys) == 0 {
		return nil, sdkerrors.ErrInvalidPubKey.Wrap("morse multisig must have at least one key")
	}

	var pubKeys []cometed.PubKey
	for _, priv := range morsePrivKeys {
		if edPub, ok := priv.PubKey().(cometed.PubKey); ok {
			pubKeys = append(pubKeys, edPub)
		} else {
			return nil, sdkerrors.ErrInvalidPubKey.Wrap("non-ed25519 key provided in multisig")
		}
	}

	msg := &MsgClaimMorseMultiSigAccount{
		ShannonDestAddress:    shannonDestAddr,
		ShannonSigningAddress: shannonSignerAddr,
		MorsePublicKeys:       pubKeys,
	}

	if err := msg.SignMsgClaimMorseMultiSigAccount(morsePrivKeys); err != nil {
		return nil, sdkerrors.ErrInvalidPubKey.Wrapf("failed to sign morse multisig: %s", err)
	}

	return msg, nil
}

// ValidateBasic performs basic checks: valid bech32 address, valid public keys, and signature.
func (msg *MsgClaimMorseMultiSigAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid Shannon destination address (%s): %s", msg.ShannonDestAddress, err)
	}

	return msg.ValidateMorseSignature()
}

// signWithMorseMultiSigKeys signs the message with all provided private keys and merges into one signature.
func (msg *MsgClaimMorseMultiSigAccount) SignMsgClaimMorseMultiSigAccount(privKeys []cometcrypto.PrivKey) error {
	signBytes, err := msg.getSigningBytes()
	if err != nil {
		return ErrMorseSignature.Wrapf("unable to get signing bytes: %s", err)
	}

	var sigs [][]byte
	for _, priv := range privKeys {
		sig, err := priv.Sign(signBytes)
		if err != nil {
			return ErrMorseSignature.Wrapf("unable to sign with one of the keys: %s", err)
		}
		sigs = append(sigs, sig)
	}

	// Concatenate all sigs (CometBFT old-style multisig expects all keys to sign)
	msg.MorseSignature = combineMultiSigSigs(sigs)
	return nil
}

// validateMorseMultiSigSignature verifies all public keys match and the message was signed correctly.
func (msg *MsgClaimMorseMultiSigAccount) ValidateMorseSignature() error {
	signBytes, err := msg.getSigningBytes()
	if err != nil {
		return ErrMorseSignature.Wrapf("failed to marshal for signature check: %s", err)
	}

	sig := msg.GetMorseSignature()

	expectedCount := len(msg.GetMorsePublicKeys())

	if len(sig) != expectedCount*MorseSignatureLengthBytes {
		return ErrMorseSignature.Wrap("multisig signature length mismatch")
	}

	for i, pub := range msg.GetMorsePublicKeys() {
		start := i * MorseSignatureLengthBytes
		end := start + MorseSignatureLengthBytes
		if !pub.VerifySignature(signBytes, sig[start:end]) {
			return ErrMorseSignature.Wrapf("invalid signature for key #%d", i)
		}
	}

	return nil
}

// getSigningBytes returns the deterministic byte encoding of the message with signature removed.
func (msg *MsgClaimMorseMultiSigAccount) getSigningBytes() ([]byte, error) {
	copyMsg := *msg
	copyMsg.MorseSignature = nil
	return proto.Marshal(&copyMsg)
}

func (msg *MsgClaimMorseMultiSigAccount) GetPublicKeyMultiSignature() *morsecrypto.PublicKeyMultiSignature {
	pks := msg.GetMorsePublicKeys()
	newKeys := make([]morsecrypto.PublicKey, len(pks))
	for i, pk := range pks {
		// Note: Support for secp256k1 is not yet implemented
		newKey, err := morsecrypto.Ed25519PublicKey{}.NewPublicKey(pk.Bytes())
		if err != nil {
			panic(err)
		}
		newKeys[i] = newKey
	}

	pms := &morsecrypto.PublicKeyMultiSignature{PublicKeys: newKeys}
	return pms
}

// GetMorseSrcAddress returns the derived address of the multisig keyset (usually hash of pubkeys).
func (msg *MsgClaimMorseMultiSigAccount) GetMorseSrcAddress() string {
	return msg.GetPublicKeyMultiSignature().Address().String()
}

// GetMorsePublicKeyBz returns the Amino-encoded public key of the given message.
func (msg *MsgClaimMorseMultiSigAccount) GetMorsePublicKeyBz() []byte {
	return msg.GetPublicKeyMultiSignature().Bytes()
}

func combineMultiSigSigs(sigs [][]byte) []byte {
	// naive: just concatenate in order
	var combined []byte
	for _, sig := range sigs {
		combined = append(combined, sig...)
	}
	return combined
}
