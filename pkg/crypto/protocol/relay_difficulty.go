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
	// If targetNumRelays == newRelaysEma -> do not scale -> keep the same difficulty to mine relays
	// TODO_IMPROVE: Figure out if there's a range (e.g. 5%) withing which it is reasonable
	// to keep the same difficulty.
	if targetNumRelays == newRelaysEma {
		return prevTargetHash
	}

	// Calculate the proportion of target relays relative to the EMA of actual volume applicable relays
	// If difficultyScalingRatio < 1 -> scale down -> increase difficulty to mine relays
	// If difficultyScalingRatio > 1 -> scale up -> decrease difficulty to mine relays
	difficultyScalingRatio := big.NewRat(int64(targetNumRelays), int64(newRelaysEma))
	scaledDifficultyHashBz := ScaleRelayDifficultyHash(prevTargetHash, difficultyScalingRatio)
	// Trim any extra zeros from the scaled hash to ensure that it only contains
	// the meaningful bytes.
	scaledDifficultyHashBz = bytes.TrimLeft(scaledDifficultyHashBz, "\x00")
	// If scaledDifficultyHash is longer than BaseRelayDifficultyHashBz, then use
	// BaseRelayDifficultyHashBz as we should not have a bigger hash than the base.
	if len(scaledDifficultyHashBz) > len(BaseRelayDifficultyHashBz) {
		return BaseRelayDifficultyHashBz
	}

	// Ensure the scaled hash is padded to (at least) the same length as the provided hash.
	if len(scaledDifficultyHashBz) < len(prevTargetHash) {
		paddingOffset := len(prevTargetHash) - len(scaledDifficultyHashBz)

		paddedScaledDifficultyHash := make([]byte, len(prevTargetHash))
		copy(paddedScaledDifficultyHash[paddingOffset:], scaledDifficultyHashBz)
		return paddedScaledDifficultyHash
	}

	return scaledDifficultyHashBz
}

// ScaleRelayDifficultyHash scales the provided hash based on the given ratio.
// If the ratio is less than 1, the hash will be scaled down.
// DEV_NOTE: Only exposed publicly for testing purposes.
func ScaleRelayDifficultyHash(
	initialDifficultyHashBz []byte,
	difficultyScalingRatio *big.Rat,
) []byte {
	difficultyHashInt := bytesToBigInt(initialDifficultyHashBz)
	difficultyHashRat := new(big.Rat).SetInt(difficultyHashInt)

	// Scale the current by multiplying it by the ratio.
	scaledHashRat := new(big.Rat).Mul(difficultyHashRat, difficultyScalingRatio)
	scaledHashInt := new(big.Int).Div(scaledHashRat.Num(), scaledHashRat.Denom())
	// Convert the scaled hash to a byte slice.
	return scaledHashInt.Bytes()
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
