package sample

import (
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/encoding"
)

// func(text string) ([]byte, error) { return sdk.AccAddressFromBech32(text) },
// func(text string) ([]byte, error) { return sdk.ValAddressFromBech32(text) },
// func(text string) ([]byte, error) { return sdk.ConsAddressFromBech32(text) },

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

// AccAddress returns a sample account address
func AccAddress() string {
	addr, _ := AccAddressAndPubKey()
	return addr
}

// ValOperatorAddress returns a sample validator operator address
func ValOperatorAddress() string {
	// Generate a new keypair for the validator operator
	sk := secp256k1.GenPrivKey()
	pk := sk.PubKey()
	addr := pk.Address()
	// Convert to validator operator address with proper prefix
	return cosmostypes.ValAddress(addr).String()
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
