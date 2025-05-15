package crypto

import (
	"encoding/hex"
	"encoding/json"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"strings"
)

type (
	Ed25519PublicKey ed25519.PubKey
)

var (
	_ PublicKey     = Ed25519PublicKey{}
	_ crypto.PubKey = Ed25519PublicKey{}
)

const (
	Ed25519PubKeySize    = ed25519.PubKeySize
	Ed25519SignatureSize = ed25519.SignatureSize
)

func (Ed25519PublicKey) NewPublicKey(b []byte) (PublicKey, error) {
	pubkey := ed25519.PubKey(b)
	pk := Ed25519PublicKey(pubkey)
	return pk, nil
}

func (Ed25519PublicKey) PubKeyToPublicKey(key crypto.PubKey) PublicKey {
	return Ed25519PublicKey(key.(ed25519.PubKey))
}

func (pub Ed25519PublicKey) PubKey() crypto.PubKey {
	return ed25519.PubKey(pub)
}

func (pub Ed25519PublicKey) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(pub)
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

func (pub Ed25519PublicKey) Address() crypto.Address {
	return ed25519.PubKey(pub).Address()
}

func (pms Ed25519PublicKey) Type() string {
	return ed25519.KeyType
}

func (pub Ed25519PublicKey) VerifySignature(msg []byte, sig []byte) bool {
	return ed25519.PubKey(pub).VerifySignature(msg, sig)
}

func (pub Ed25519PublicKey) VerifyBytes(msg []byte, sig []byte) bool {
	return ed25519.PubKey(pub).VerifySignature(msg, sig)
}

func (pub Ed25519PublicKey) Equals(other crypto.PubKey) bool {
	return ed25519.PubKey(pub).Equals(ed25519.PubKey(other.(Ed25519PublicKey)))
}

func (pub Ed25519PublicKey) Size() int {
	return Ed25519PubKeySize
}

func (pub Ed25519PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pub.RawString())
}

func (pub *Ed25519PublicKey) UnmarshalJSON(data []byte) error {
	hexstring := strings.Trim(string(data[:]), "\"")

	bytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return err
	}
	pk, err := NewPublicKeyBz(bytes)
	if err != nil {
		return err
	}
	err = cdc.UnmarshalBinaryBare(pk.Bytes(), pub)

	if err != nil {
		return err
	}

	return nil
}
