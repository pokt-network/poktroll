package sample

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccAddressAndPubKey returns a sample account address and public key
func AccAddressAndPubKey() (string, cryptotypes.PubKey) {
	pk := secp256k1.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String(), pk
}

// AccAddress returns a sample account address
func AccAddress() string {
	addr, _ := AccAddressAndPubKey()
	return addr
}

// AccAddressAndPubKeyEdd2519 returns a sample account address and public key
func AccAddressAndPubKeyEdd2519() (string, cryptotypes.PubKey) {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String(), pk
}
