package protocol

import (
	"bytes"
	"encoding/hex"
	"math/big"
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
func IsRelayVolumeApplicable(relayHash, targetHash []byte) bool {
	return bytes.Compare(relayHash, targetHash) == -1 // True if relayHash < targetHash
}

// ComputeNewDifficultyTargetHash computes the new difficulty target hash based
// on the target number of relays we want the network to mine and the new EMA of
// the number of relays.
func ComputeNewDifficultyTargetHash(prevTargetHash []byte, targetNumRelays, newRelaysEma uint64) []byte {
	// The target number of relays we want the network to mine is greater than
	// the actual volume applicable relays.
	// Default to the baseline relay mining difficulty (no scaling necessary)
	if targetNumRelays > newRelaysEma {
		return BaseRelayDifficultyHashBz
	}

	// Calculate the proportion of target relays relative to the EMA of actual volume applicable relays
	// TODO_POST_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	ratio := new(big.Float).Quo(
		new(big.Float).SetUint64(targetNumRelays),
		new(big.Float).SetUint64(newRelaysEma),
	)

	// Compute the new target hash by scaling the previous target hash based on the ratio
	return ScaleRelayDifficultyHash(prevTargetHash, ratio)
}

// ScaleRelayDifficultyHash scales the current hash based on the given ratio.
func ScaleRelayDifficultyHash(currHashBz []byte, ratio *big.Float) []byte {
	// Convert currentHash to a big.Float to minimize precision loss.
	// TODO_POST_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	currHashInt := bytesToBigInt(currHashBz)
	currHashFloat := new(big.Float).SetInt(currHashInt)

	// Scale the current by multiplying it by the ratio.
	scaledHashFloat := new(big.Float).Mul(currHashFloat, ratio)
	// NB: Some precision is lost when converting back to an integer.
	scaledHashInt, _ := scaledHashFloat.Int(nil)
	scaledHashBz := scaledHashInt.Bytes()

	// Ensure the scaled current hash maxes out at BaseRelayDifficulty
	if len(scaledHashBz) > len(currHashBz) {
		return BaseRelayDifficultyHashBz
	}

	// Ensure the scaled current hash has the same length as the default current hash.
	if len(scaledHashBz) < len(currHashBz) {
		paddedHash := make([]byte, len(currHashBz))
		copy(paddedHash[len(paddedHash)-len(scaledHashBz):], scaledHashBz)
		return paddedHash
	}

	return scaledHashBz
}

// GetRelayDifficultyProbability returns a fraction that determines the probability that a
// target (i.e. difficulty) hash is relative to the baseline.
func GetRelayDifficultyProbability(relayDifficultyHash []byte) *big.Rat {
	target := bytesToBigInt(relayDifficultyHash)
	maxHash := bytesToBigInt(BaseRelayDifficultyHashBz)
	probability := new(big.Rat).SetFrac(target, maxHash)
	return probability
}

// GetRelayDifficultyMultiplier returns the inverse of GetRelayDifficultyProbability
// to scale on-chain volume applicable relays to estimated serviced off-chain relays.
func GetRelayDifficultyMultiplier(relayDifficultyHash []byte) *big.Rat {
	probability := GetRelayDifficultyProbability(relayDifficultyHash)
	return new(big.Rat).Inv(probability)
}

// GetRelayDifficultyMultiplierUInt returns a human readable integer representation
// of GetRelayDifficultyMultiplier for telemetry purposes.
// TODO_BETA(@red-0ne): Refactor this function to avoid using ints for both
// telemetry and business logic. Use Rat in business logic and float32 for telemetry.
// Ref: https://github.com/pokt-network/poktroll/pull/771#discussion_r1761517063
func GetRelayDifficultyMultiplierUInt(relayDifficultyHash []byte) uint64 {
	ratToUint64 := func(rat *big.Rat) uint64 {
		float, _ := rat.Float64()
		return uint64(float)
	}
	probability := GetRelayDifficultyProbability(relayDifficultyHash)
	return ratToUint64(probability)
}

// Convert byte slice to a big integer
func bytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}
