package sample

import (
	"encoding/hex"

	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// AccAddressAndPubKey returns a sample account address and public key
func AccAddressAndPubKey() (string, cryptotypes.PubKey) {
	pk := secp256k1.GenPrivKey().PubKey()
	addr := pk.Address()
	return cosmostypes.AccAddress(addr).String(), pk
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
	return cosmostypes.AccAddress(addr).String(), pk
}

// ConsAddress returns a sample consensus node address, which has the prefix
// of consensus nodes when converted to bech32. Consensus addresses identify
// the validator node in the consensus engine and are derived using ed25519.
// See: https://docs.cosmos.network/main/learn/beginner/accounts#addresses
func ConsAddress() cosmostypes.ConsAddress {
	_, pk := AccAddressAndPubKeyEd25519()
	consensusAddress := tmhash.SumTruncated(pk.Address())
	consAddress := cosmostypes.ConsAddress(consensusAddress)
	return consAddress
}

// ConsAddressBech32 returns a bech32-encoded  sample consensus node address,
// which has the prefix of consensus nodes when converted to bech32. Consensus
// addresses identify the validator node in the consensus engine and are derived
// using ed25519.
// See: https://docs.cosmos.network/main/learn/beginner/accounts#addresses
func ConsAddressBech32() string {
	return ConsAddress().String()
}

// AccAddressFromConsBech32 returns an account address (with the Bech32PrefixForAccount prefix)
// from a given consensus address (with the Bech32PrefixForConsensus prefix).
//
// Reference: see initSDKConfig in  `cmd/poktrolld/cmd`.
//
// Use case: in the native cosmos SDK mint module, we set inflation_rate_change to 0
// because Pocket Network has a custom inflation mechanism. Data availability and
// block validation is a small part of the network's utility, so the majority of
// inflation comes from the relays serviced. Therefore, the validator's (block producer's)
// rewards are proportional to that as well. For this reason, we need a helper function
// to identify the proposer address from its validator consensus address (ed255519).
func AccAddressFromConsBech32(consBech32 string) string {
	consAccAddr, _ := cosmostypes.ConsAddressFromBech32(consBech32)
	accAddr, _ := cosmostypes.AccAddressFromHexUnsafe(hex.EncodeToString(consAccAddr.Bytes()))
	return accAddr.String()
}
