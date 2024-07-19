package protocol

import (
	"bytes"
	"encoding/hex"
	"math/big"
)

var (
	// BaseRelayDifficultyHashBz is the chosen "highest" (easiest) target hash, which
	// corresponds to the lowest possible difficulty.
	//
	// It effectively normalizes the difficulty number (which is returned by GetDifficultyFromHash)
	// by defining the hash which corresponds to the base difficulty.
	//
	// When this is the difficulty of a particular service, all relays are reward / volume applicable.
	//
	// Bitcoin uses a similar concept, where the target hash is defined as the hash:
	// - https://bitcoin.stackexchange.com/questions/107976/bitcoin-difficulty-why-leading-0s
	// - https://bitcoin.stackexchange.com/questions/121920/is-it-always-possible-to-find-a-number-whose-hash-starts-with-a-certain-number-o
	BaseRelayDifficultyHashBz, _ = hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

// GetDifficultyFromHash returns the "difficulty" of the given hash, with respect
// to the "highest" (easiest) target hash, BaseRelayDifficultyHash.
// The resultant value is not used for any business logic but is simplify there to have a human-readable version of the hash.
func GetDifficultyFromHash(hashBz [RelayHasherSize]byte) int64 {
	baseRelayDifficultyHashInt := new(big.Int).SetBytes(BaseRelayDifficultyHashBz)
	hashInt := new(big.Int).SetBytes(hashBz[:])

	// difficulty is the ratio of the highest target hash to the given hash.
	return new(big.Int).Div(baseRelayDifficultyHashInt, hashInt).Int64()
}

// IsRelayVolumeApplicable returns true if the relay IS reward / volume applicable.
// A relay is reward / volume applicable IFF its hash is less than the target hash.
//   - relayHash is the hash of the relay to be checked.
//   - targetHash is the hash of the relay difficulty target for a particular service.
//
// TODO_MAINNET: Devise a test that tries to attack the network and ensure that
// there is sufficient telemetry.
func IsRelayVolumeApplicable(relayHash, targetHash []byte) bool {
	return bytes.Compare(relayHash, targetHash) == -1 // True if relayHash < targetHash
}
