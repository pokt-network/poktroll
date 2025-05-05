package types

import (
	"encoding/hex"
	cometcrypto "github.com/cometbft/cometbft/crypto"
	cometed "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
	amino "github.com/tendermint/go-amino"
)

var (
	_ sdk.Msg                   = (*MsgClaimMorseMultisigAccount)(nil)
	_ morseMultisigClaimMessage = (*MsgClaimMorseMultisigAccount)(nil)
)

// NewMsgClaimMorseMultisigAccount constructs a new MsgClaimMorseMultisigAccount and signs it if privKeys are provided.
func NewMsgClaimMorseMultisigAccount(
	shannonDestAddr string,
	morsePrivKeys []cometcrypto.PrivKey,
	shannonSignerAddr string,
) (*MsgClaimMorseMultisigAccount, error) {

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

	msg := &MsgClaimMorseMultisigAccount{
		ShannonDestAddress:      shannonDestAddr,
		ShannonSigningAddress:   shannonSignerAddr,
		MorseMultisigPublicKeys: pubKeys,
	}

	if err := msg.SignMsgClaimMorseMultisigAccount(morsePrivKeys); err != nil {
		return nil, sdkerrors.ErrInvalidPubKey.Wrapf("failed to sign morse multisig: %s", err)
	}

	return msg, nil
}

// ValidateBasic performs basic checks: valid bech32 address, valid public keys, and signature.
func (msg *MsgClaimMorseMultisigAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid Shannon destination address (%s): %s", msg.ShannonDestAddress, err)
	}

	return msg.ValidateMorseSignature()
}

// signWithMorseMultisigKeys signs the message with all provided private keys and merges into one signature.
func (msg *MsgClaimMorseMultisigAccount) SignMsgClaimMorseMultisigAccount(privKeys []cometcrypto.PrivKey) error {
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
	msg.MorseSignature = combineMultisigSigs(sigs)
	return nil
}

// validateMorseMultisigSignature verifies all public keys match and the message was signed correctly.
func (msg *MsgClaimMorseMultisigAccount) ValidateMorseSignature() error {
	return validateMorseMultisigSignature(msg)
}

// getSigningBytes returns the deterministic byte encoding of the message with signature removed.
func (msg *MsgClaimMorseMultisigAccount) getSigningBytes() ([]byte, error) {
	copyMsg := *msg
	copyMsg.MorseSignature = nil
	return proto.Marshal(&copyMsg)
}

// GetMorseSrcAddress returns the derived address of the multisig keyset (usually hash of pubkeys).
func (msg *MsgClaimMorseMultisigAccount) GetMorseSrcAddress() string {
	return deriveMultisigAddress(msg.MorseMultisigPublicKeys)
}

func combineMultisigSigs(sigs [][]byte) []byte {
	// naive: just concatenate in order
	var combined []byte
	for _, sig := range sigs {
		combined = append(combined, sig...)
	}
	return combined
}

func validateMorseMultisigSignature(msg morseMultisigClaimMessage) error {
	signBytes, err := msg.getSigningBytes()
	if err != nil {
		return sdkerrors.ErrUnauthorized.Wrapf("failed to marshal for signature check: %s", err)
	}

	sig := msg.GetMorseSignature()
	expectedCount := len(msg.GetMorseMultisigPublicKeys())

	if len(sig) != expectedCount*MorseSignatureLengthBytes {
		return sdkerrors.ErrUnauthorized.Wrap("multisig signature length mismatch")
	}

	for i, pub := range msg.GetMorseMultisigPublicKeys() {
		start := i * MorseSignatureLengthBytes
		end := start + MorseSignatureLengthBytes
		if !pub.VerifySignature(signBytes, sig[start:end]) {
			return sdkerrors.ErrUnauthorized.Wrapf("invalid signature for key #%d", i)
		}
	}

	return nil
}

type PublicKey interface {
	PubKey() cometcrypto.PubKey
	Bytes() []byte
	RawBytes() []byte
	String() string
	RawString() string
	Address() cometcrypto.Address
	Equals(other cometcrypto.PubKey) bool
	VerifyBytes(msg []byte, sig []byte) bool
	PubKeyToPublicKey(cometcrypto.PubKey) PublicKey
	Size() int
}

// Minimal copy of PublicKeyMultiSignature
type PublicKeyMultiSignature struct {
	PublicKeys []PublicKey `json:"keys"`
}

var multisigCodec = func() *amino.Codec {
	cdc := amino.NewCodec()
	cdc.RegisterInterface((*PublicKey)(nil), nil)
	cdc.RegisterConcrete(PublicKeyMultiSignature{}, "crypto/public_key_multi_signature", nil)
	cdc.RegisterConcrete(Ed25519PublicKey{}, "crypto/ed25519_public_key", nil)
	return cdc
}()

func deriveMultisigAddress(pks []cometed.PubKey) string {
	newKeys := make([]PublicKey, len(pks))
	for i, pk := range pks {
		newKeys[i] = Ed25519PublicKey(pk)
	}

	pms := PublicKeyMultiSignature{PublicKeys: newKeys}
	encoded, err := multisigCodec.MarshalBinaryBare(pms)
	if err != nil {
		panic(sdkerrors.ErrInvalidPubKey.Wrapf("failed to encode PublicKeyMultiSignature: %s", err))
	}

	return cometcrypto.AddressHash(encoded).String()
}

type (
	Ed25519PublicKey cometed.PubKey
)

var (
	_ PublicKey = Ed25519PublicKey{}
)

const (
	Ed25519PubKeySize = cometed.PubKeySize
)

func (Ed25519PublicKey) FromBytes(b []byte) (PublicKey, error) {
	return Ed25519PublicKey(cometed.PubKey(b)), nil
}

func (Ed25519PublicKey) PubKeyToPublicKey(key cometcrypto.PubKey) PublicKey {
	return Ed25519PublicKey(key.(cometed.PubKey))
}

func (pub Ed25519PublicKey) PubKey() cometcrypto.PubKey {
	return cometed.PubKey(pub)
}

func (pub Ed25519PublicKey) Bytes() []byte {
	bz, err := multisigCodec.MarshalBinaryBare(pub)
	if err != nil {
		panic(err)
	}
	return bz
}

func (pub Ed25519PublicKey) RawBytes() []byte {
	pkBytes := [Ed25519PubKeySize]byte(pub)
	return pkBytes[:]
}

func (pub Ed25519PublicKey) String() string {
	return hex.EncodeToString(pub.Bytes())
}

func (pub Ed25519PublicKey) RawString() string {
	return hex.EncodeToString(pub.RawBytes())
}

func (pub Ed25519PublicKey) Address() cometcrypto.Address {
	return cometed.PubKey(pub).Address()
}

func (pub Ed25519PublicKey) VerifyBytes(msg []byte, sig []byte) bool {
	return cometed.PubKey(pub).VerifySignature(msg, sig)
}

func (pub Ed25519PublicKey) Equals(other cometcrypto.PubKey) bool {
	return cometed.PubKey(pub).Equals(other)
}

func (pub Ed25519PublicKey) Size() int {
	return Ed25519PubKeySize
}
