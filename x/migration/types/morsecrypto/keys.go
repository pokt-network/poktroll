package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
)

type PublicKey interface {
	PubKey() crypto.PubKey
	Bytes() []byte
	RawBytes() []byte
	String() string
	RawString() string
	Address() crypto.Address
	Equals(other crypto.PubKey) bool
	VerifyBytes(msg []byte, sig []byte) bool
	PubKeyToPublicKey(crypto.PubKey) PublicKey
	Size() int
}

type PublicKeyMultiSig interface {
	Address() crypto.Address
	String() string
	Bytes() []byte
	Equals(other crypto.PubKey) bool
	VerifyBytes(msg []byte, multiSignature []byte) bool
	PubKey() crypto.PubKey
	RawBytes() []byte
	RawString() string
	PubKeyToPublicKey(crypto.PubKey) PublicKey
	Size() int
	// new methods
	NewMultiKey(keys ...PublicKey) (PublicKeyMultiSig, error)
	Keys() []PublicKey
}

type MultiSig interface {
	AddSignature(sig []byte, key PublicKey, keys []PublicKey) (MultiSig, error)
	AddSignatureByIndex(sig []byte, index int) MultiSig
	Marshal() []byte
	Unmarshal([]byte) MultiSig
	NewMultiSignature() MultiSig
	String() string
	NumOfSigs() int
	Signatures() [][]byte
	GetSignatureByIndex(i int) (sig []byte, found bool)
}

func NewPublicKey(hexString string) (PublicKey, error) {
	b, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return NewPublicKeyBz(b)
}

func NewPublicKeyBz(b []byte) (PublicKey, error) {
	x := len(b)
	if x == Ed25519PubKeySize {
		return Ed25519PublicKey{}.NewPublicKey(b)
	} else if pk, err := PublicKeyMultiSignature.NewPublicKey(PublicKeyMultiSignature{}, b); err == nil {
		return pk, err
	} else {
		return nil, fmt.Errorf("unsupported public key type, length of: %d", x)
	}
}

func PubKeyToPublicKey(key crypto.PubKey) (PublicKey, error) {
	k := key
	switch k.(type) {
	case ed25519.PubKey:
		return Ed25519PublicKey(key.(ed25519.PubKey)), nil
	case Ed25519PublicKey:
		return key.(Ed25519PublicKey), nil
	default:
		return nil, errors.New("error converting pubkey to public key -> unsupported public key type")
	}
}

func PubKeyFromBytes(pubKeyBytes []byte) (pubKey PublicKey, err error) {
	err = cdc.UnmarshalBinaryBare(pubKeyBytes, &pubKey)
	return
}
