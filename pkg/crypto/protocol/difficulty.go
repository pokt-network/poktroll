package protocol

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	// BaseRelayDifficultyHashBz is the chosen "highest" (easiest) target hash,
	// which corresponds to the lowest possible difficulty.
	//
	// In simple terms, it mean "every relay is a volume applicable relay".
	//
	// It other words, it is used to normalize all relay mining difficulties.
	//
	// Bitcoin uses a similar concept, where the target hash is defined as the hash:
	// - https://bitcoin.stackexchange.com/questions/107976/bitcoin-difficulty-why-leading-0s
	// - https://bitcoin.stackexchange.com/questions/121920/is-it-always-possible-to-find-a-number-whose-hash-starts-with-a-certain-number-o
	BaseRelayDifficultyHashHex   = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // all relays are payable
	BaseRelayDifficultyHashBz, _ = hex.DecodeString(BaseRelayDifficultyHashHex)
)

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

// ComputeNewDifficultyTargetHash computes the new difficulty target hash based
// on the target number of relays we want the network to mine and the new EMA of
// the number of relays.
// NB: Exported for testing purposes only.
func ComputeNewDifficultyTargetHash(prevTargetHash []byte, targetNumRelays, newRelaysEma uint64) []byte {
	// The target number of relays we want the network to mine is greater than
	// the actual on-chain relays, so we don't need to scale to anything above
	// the default.
	if targetNumRelays > newRelaysEma {
		return BaseRelayDifficultyHashBz
	}

	// Calculate the proportion of target relays relative to the EMA of actual volume applicable relays
	// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	ratio := new(big.Float).Quo(
		new(big.Float).SetUint64(targetNumRelays),
		new(big.Float).SetUint64(newRelaysEma),
	)

	// Compute the new target hash by scaling the previous target hash based on the ratio
	newTargetHash := scaleDifficultyTargetHash(prevTargetHash, ratio)

	return newTargetHash
}

// scaleDifficultyTargetHash scales the target hash based on the given ratio.
func scaleDifficultyTargetHash(targetHash []byte, ratio *big.Float) []byte {
	// Convert targetHash to a big.Float to minimize precision loss.
	targetInt := bytesToBigInt(targetHash)
	// TODO_POST_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	targetFloat := new(big.Float).SetInt(targetInt)

	// Scale the target by multiplying it by the ratio.
	scaledTargetFloat := new(big.Float).Mul(targetFloat, ratio)
	// NB: Some precision is lost when converting back to an integer.
	scaledTargetInt, _ := scaledTargetFloat.Int(nil)
	scaledTargetHash := scaledTargetInt.Bytes()

	// Ensure the scaled target hash maxes out at BaseRelayDifficulty
	if len(scaledTargetHash) > len(targetHash) {
		return BaseRelayDifficultyHashBz
	}

	// Ensure the scaled target hash has the same length as the default target hash.
	if len(scaledTargetHash) < len(targetHash) {
		paddedTargetHash := make([]byte, len(targetHash))
		copy(paddedTargetHash[len(paddedTargetHash)-len(scaledTargetHash):], scaledTargetHash)
		return paddedTargetHash
	}

	return scaledTargetHash
}

// GetDifficultyFromHash returns the "difficulty" of the given hash, with respect
// to the "highest" (easiest) target hash, BaseRelayDifficultyHash.
// The resultant value is not used for any business logic but is simplify there to have a human-readable version of the hash.
// TODO_MAINNET: Can this cause an integer overflow?
func GetDifficultyFromHash(hashBz [RelayHasherSize]byte) int64 {
	baseRelayDifficultyHashInt := bytesToBigInt(BaseRelayDifficultyHashBz)
	hashInt := bytesToBigInt(hashBz[:])

	// difficulty is the ratio of the highest target hash to the given hash.
	// TODO_MAINNET: Can this cause an integer overflow?
	return new(big.Int).Div(baseRelayDifficultyHashInt, hashInt).Int64()
}

// RelayDifficultyProbability returns a fraction that determines the probability that a
// target (i.e. difficulty) hash is relative to the baseline.
func RelayDifficultyProbability(targetRelayDifficulty []byte) *big.Rat {
	target := bytesToBigInt(targetRelayDifficulty)
	maxHash := bytesToBigInt(BaseRelayDifficultyHashBz)
	probability := new(big.Rat).SetFrac(target, maxHash)
	return probability
}

func RelayDifficultyMultiplier(targetRelayDifficulty []byte) *big.Rat {
	probability := RelayDifficultyProbability(targetRelayDifficulty)
	return new(big.Rat).Inv(probability)
}

func ScaleRelays(t *testing.T, relayDifficultyProbability *big.Rat, numRelays uint64) uint64 {
	mr := new(big.Rat).SetUint64(numRelays)
	result := new(big.Rat).Mul(relayDifficultyProbability, mr)
	num := result.Num()
	denom := result.Denom()
	quotient := new(big.Int).Div(num, denom)
	require.True(t, quotient.IsUint64(), "value out of range for uint64")
	return quotient.Uint64()
}

func ratToUint64(t *testing.T, relayDifficultyProbability *big.Rat) uint64 {
	num := relayDifficultyProbability.Num()
	denom := relayDifficultyProbability.Denom()
	quotient := new(big.Int).Div(num, denom)
	require.True(t, quotient.IsUint64(), "value out of range for uint64")
	return quotient.Uint64()
}

// Convert byte slice to a big integer
func bytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}
