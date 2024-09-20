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
	// TODO_MAINNET(@red-0ne): Use a language agnostic float implementation to ensure
	// deterministic results and avoid loss of precision. Specifically, we need to
	// use big.Rat, delay any computation.

	// Calculate the proportion of target relays relative to the EMA of actual volume applicable relays
	difficultyScalingRatio := new(big.Float).Quo(
		new(big.Float).SetUint64(targetNumRelays),
		new(big.Float).SetUint64(newRelaysEma),
	)
	// If difficultyScalingRatio < 1 -> scale down -> increase difficulty to mine relays
	// If difficultyScalingRatio > 1 -> scale up -> decrease difficulty to mine relays
	if difficultyScalingRatio.Cmp(big.NewFloat(1)) == 0 {
		return prevTargetHash
	}

	// You can't scale the base relay difficulty hash below BaseRelayDifficultyHashBz
	isDecreasingDifficulty := difficultyScalingRatio.Cmp(big.NewFloat(1)) == 1
	if isDecreasingDifficulty && bytes.Equal(prevTargetHash, BaseRelayDifficultyHashBz) {
		return BaseRelayDifficultyHashBz
	}

	scaledDifficultyHash := ScaleRelayDifficultyHash(prevTargetHash, difficultyScalingRatio)
	if len(scaledDifficultyHash) > len(BaseRelayDifficultyHashBz) {
		return BaseRelayDifficultyHashBz
	}

	// Compute the new target hash by scaling the previous target hash based on the ratio
	return ScaleRelayDifficultyHash(prevTargetHash, difficultyScalingRatio)
}

// ScaleRelayDifficultyHash scales the provided hash based on the given ratio.
// If the ratio is less than 1, the hash will be scaled down.
// DEV_NOTE: Only exposed publicly for testing purposes.
func ScaleRelayDifficultyHash(difficultyHashBz []byte, ratio *big.Float) []byte {
	// Convert difficultyHashBz to a big.Float to minimize precision loss.
	// TODO_MAINNET(@red-one): Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	prevHashInt := bytesToBigInt(difficultyHashBz)
	prevHashFloat := new(big.Float).SetInt(prevHashInt)

	// Scale the current by multiplying it by the ratio.
	// TODO(@red-0ne): Ensure that the precision lost here doesn't cause any
	// major issues by using big.Rat.
	scaledHashFloat := new(big.Float).Mul(prevHashFloat, ratio)
	scaledHashInt, _ := scaledHashFloat.Int(nil)
	// scaledHashBz := make([]byte, len(BaseRelayDifficultyHashBz))
	// scaledHashInt.FillBytes(scaledHashBz)
	scaledHashBz := scaledHashInt.Bytes()

	// Ensure the scaled hash is padded to (at least) the same length as the provided hash.
	if len(scaledHashBz) < len(difficultyHashBz) {
		paddedHash := make([]byte, len(difficultyHashBz))
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

// GetRelayDifficultyMultiplierToFloat32 returns a human readable integer
// representation of GetRelayDifficultyProbability.
// THIS IS TO BE USED FOR TELEMETRY PURPOSES ONLY.
// See the following discussing for why we're using a float32:
// https://github.com/pokt-network/poktroll/pull/771#discussion_r1761517063
func GetRelayDifficultyMultiplierToFloat32(relayDifficultyHash []byte) float32 {
	ratToFloat32 := func(rat *big.Rat) float32 {
		floatValue, _ := rat.Float32()
		return floatValue
	}
	probability := GetRelayDifficultyProbability(relayDifficultyHash)
	return ratToFloat32(probability)
}

// Convert byte slice to a big integer
func bytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}
