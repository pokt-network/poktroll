package sample

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccAddress returns a sample account address
func AccAddress() string {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String()
}

// AccPubKey returns a sample account public key
func AccPubKey() cryptotypes.PubKey {
	return ed25519.GenPrivKey().PubKey()
}

// AddrAndPubKey returns a sample account address and public key
func AddrAndPubKey() (string, cryptotypes.PubKey) {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String(), pk
}
