package sample

import (
	"github.com/cometbft/cometbft/crypto/tmhash"
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
	// TODO_BETA(@olshansk): Change this to secp256k1 because that's what we'll
	// use in production for all real accounts.
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String()
}

// ConsAddress returns a sample consensus address, which has the prefix
// of validators (i.e. consensus nodes) when converted to bech32.
func ConsAddress() string {
	pk := ed25519.GenPrivKey().PubKey()
	consensusAddress := tmhash.SumTruncated(pk.Address())
	valAddress := sdk.ValAddress(consensusAddress)
	return valAddress.String()
}

// AccAddressAndPubKeyEdd2519 returns a sample account address and public key
func AccAddressAndPubKeyEdd2519() (string, cryptotypes.PubKey) {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String(), pk
}
