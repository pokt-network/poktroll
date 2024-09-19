package sample

import (
	"encoding/hex"

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
	addr, _ := AccAddressAndPubKey()
	return addr
}

// AccAddressAndPubKeyEd25519 returns a sample account address and public key
func AccAddressAndPubKeyEd25519() (string, cryptotypes.PubKey) {
	pk := ed25519.GenPrivKey().PubKey()
	addr := pk.Address()
	return sdk.AccAddress(addr).String(), pk
}

// ValAddress returns a sample validator address, which has the prefix
// of validators when converted to bech32.
func ValAddress() string {
	_, pk := AccAddressAndPubKey()
	validatorAddress := tmhash.SumTruncated(pk.Address())
	valAddress := sdk.ValAddress(validatorAddress)
	return valAddress.String()
}

// ConsAddress returns a sample consensus node address, which has the prefix
// of consensus nodes when converted to bech32.
func ConsAddress() string {
	_, pk := AccAddressAndPubKey()
	consensusAddress := tmhash.SumTruncated(pk.Address())
	valAddress := sdk.ConsAddress(consensusAddress)
	return valAddress.String()
}

// AccAddressFromConsAddress returns an account address (with the Bech32PrefixForAccount prefix)
// from a given consensus address (with the Bech32PrefixForValidator prefix).
//
// Reference: see initSDKConfig in  `cmd/poktrolld/cmd`.
//
// Use case: in the native cosmos SDK mint module, we set inflation_rate_change to 0
// because Pocket Network has a custom inflation mechanism. Data availability and
// block validation is a small part of the network's utility, so the majority of
// inflation comes from the relays serviced. Therefore, the validator's (block producer's)
// rewards are proportional to that as well. For this reason, we need a helper function
// to identify the proposer address from the validator consensus address.
//
// TODO_MAINNET: Add E2E tests to validate this works as expected.
func AccAddressFromConsAddress(validatorConsAddr string) string {
	valAddr, _ := sdk.ValAddressFromBech32(validatorConsAddr)
	proposerAddress, _ := sdk.AccAddressFromHexUnsafe(hex.EncodeToString(valAddr.Bytes()))
	return proposerAddress.String()
}
