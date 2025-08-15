package sample

import (
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/encoding"
)

// AccAddressAndKeyPair returns a sample account address its public key and private key
func AccAddressAndKeyPair() (string, cryptotypes.PubKey, cryptotypes.PrivKey) {
	sk := secp256k1.GenPrivKey()
	pk := sk.PubKey()
	addr := pk.Address()
	return cosmostypes.AccAddress(addr).String(), pk, sk
}

// AccAddressAndPubKey returns a sample account address and public key
func AccAddressAndPubKey() (string, cryptotypes.PubKey) {
	address, pubKey, _ := AccAddressAndKeyPair()
	return address, pubKey
}

// AccAddressBech32 returns a sample account address
func AccAddressBech32() string {
	addr, _ := AccAddressAndPubKey()
	return addr
}

func ValOperatorAddress() cosmostypes.ValAddress {
	sk := secp256k1.GenPrivKey()
	pk := sk.PubKey()
	addr := pk.Address()
	return cosmostypes.ValAddress(addr)
}

// ValOperatorAddressBech32 returns a sample validator bech32 operator address.
func ValOperatorAddressBech32() string {
	return ValOperatorAddress().String()
}

// ConsAddress returns a sample consensus node address
func ConsAddress() cosmostypes.ConsAddress {
	privKey := ed25519.GenPrivKey()
	return cosmostypes.GetConsAddress(privKey.PubKey())
}

// ConsAddressBech32 returns a bech32-encoded consensus address
func ConsAddressBech32() string {
	return ConsAddress().String()
}

// MorseAddressHex returns the hex-encoded string representation of the address
// corresponding to a random Morse (ed25519) keypair.
func MorseAddressHex() string {
	return encoding.NormalizeMorseAddress(hex.EncodeToString(ConsAddress().Bytes()))
}
