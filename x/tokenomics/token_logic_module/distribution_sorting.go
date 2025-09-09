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

		// Append to reward data to a slice whose order is deterministic
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
func sortAddressesByFractionDesc(
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) []string {
	// Use consolidated calculation to get reward data for all addresses
	rewardData := calculateAddressRewards(stakeAmounts, totalBondedTokens, totalRewardAmount)

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

// sortAddressesByStakeDesc sorts addresses by stake amount (descending).
// Uses lexicographical address ordering as tie-breaker for determinism.
func sortAddressesByStakeDesc(stakeAmounts map[string]math.Int) []string {
	type addressStake struct {
		address string
		stake   math.Int
	}
	addressStakes := make([]addressStake, 0, len(stakeAmounts))

	// Build a slice of addressStake pairs for sorting
	for addr, stake := range stakeAmounts {
		addressStakes = append(addressStakes, addressStake{addr, stake})
	}

	// Sorting to ensure onchain behavior is deterministic:
	// Sort by:
	// 1. Stake (descending value)
	// 2. Address (ascending lexicographical order)
	sort.Slice(addressStakes, func(i, j int) bool {
		// Tie-breaker: lexicographical address order
		if addressStakes[i].stake.Equal(addressStakes[j].stake) {
			return addressStakes[i].address < addressStakes[j].address
		}
		// Descending (largest stake first)
		return addressStakes[i].stake.GT(addressStakes[j].stake)
	})

	// Extract sorted addresses
	sortedAddressesByStake := make([]string, len(addressStakes))
	for i, addrStake := range addressStakes {
		sortedAddressesByStake[i] = addrStake.address
	}

	return sortedAddressesByStake
}
