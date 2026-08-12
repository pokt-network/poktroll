package token_logic_module

// This file contains helpers to sort distribution amounts and ensure
// the results are deterministic.

import (
	"math/big"
	"sort"

	"cosmossdk.io/math"
)

// addressRewardData holds the calculated reward information for a single address.
type addressRewardData struct {
	address    string
	stake      math.Int
	baseReward math.Int
	fraction   *big.Rat
}

// calculateAddressRewards calculates both base rewards and fractional remainders for all addresses.
func calculateAddressRewards(
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) []addressRewardData {
	rewardData := make([]addressRewardData, 0, len(stakeAmounts))

	for addrStr, stake := range stakeAmounts {
		// Calculate exact proportional reward using big.Rat for precision
		// Formula: reward = (stake × totalRewardAmount) / totalBondedTokens
		exactReward := new(big.Rat).SetFrac(
			new(big.Int).Mul(stake.BigInt(), totalRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Extract integer portion as base reward
		baseReward := new(big.Int).Quo(exactReward.Num(), exactReward.Denom())
		baseRewardInt := math.NewIntFromBigInt(baseReward)

		// Calculate fractional remainder
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactReward, baseRat)

		// DEV_NOTE: stakeAmounts is a Go map, so the ORDER of this slice is NOT
		// deterministic. Consumers MUST NOT depend on it. Both current consumers are
		// order-independent by construction:
		//   - calculateBaseProportionalRewards folds it into a map keyed by address.
		//   - sortAddressesByFractionDesc sorts it with a comparator whose address
		//     tie-breaker makes the ordering total (addresses are unique map keys),
		//     so the unstable sort.Slice still yields one unique result.
		// A consumer that reads this slice in order would be consensus-breaking.
		rewardData = append(rewardData, addressRewardData{
			address:    addrStr,
			stake:      stake,
			baseReward: baseRewardInt,
			fraction:   fractionalPart,
		})
	}

	return rewardData
}

// sortAddressesByFractionDesc sorts addresses by fractional remainder (descending) for LRM.
// Addresses with largest fractional parts receive remainder tokens first.
// Uses address as ordering tie-breaker for determinism.
//
// It consumes the reward data already computed by the caller (see
// calculateProportionalRewards) rather than recomputing it: calculateAddressRewards is a
// pure function of its inputs, so recomputing here just repeated ~len(stakeholders) big.Rat
// operations per settlement for no benefit.
func sortAddressesByFractionDesc(rewardData []addressRewardData) []string {
	// Filter addresses with non-zero fractional parts
	var rewardDataNonZeroFractions []addressRewardData
	for _, data := range rewardData {
		if data.fraction.Sign() > 0 {
			rewardDataNonZeroFractions = append(rewardDataNonZeroFractions, data)
		}
	}

	// Sorting to ensure onchain behavior is deterministic:
	// Sort by:
	// 1. Fraction (descending value)
	// 2. Address (ascending lexicographical order)
	sort.Slice(rewardDataNonZeroFractions, func(i, j int) bool {
		cmp := rewardDataNonZeroFractions[i].fraction.Cmp(rewardDataNonZeroFractions[j].fraction)
		// Tie-breaker: lexicographical address order
		if cmp == 0 {
			return rewardDataNonZeroFractions[i].address < rewardDataNonZeroFractions[j].address
		}
		// Descending (largest fractions first)
		return cmp > 0
	})

	// Extract sorted addresses
	var sortedAddressesWithNonZeroFractions []string
	for _, af := range rewardDataNonZeroFractions {
		sortedAddressesWithNonZeroFractions = append(sortedAddressesWithNonZeroFractions, af.address)
	}

	return sortedAddressesWithNonZeroFractions
}
