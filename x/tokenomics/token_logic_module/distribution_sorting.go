package token_logic_module

// This file contains helpers to sort distribution amounts and ensure
// the results are deterministic.

import (
	"math/big"
	"sort"

	"cosmossdk.io/math"
)

// sortAddressesByFractionDesc sorts addresses by fractional remainder (descending) for LRM.
// Addresses with largest fractional parts receive remainder tokens first.
// Uses address as tie-breaker for determinism.
// TODO_TEST: Add test case verifying deterministic LRM distribution with equal fractions
func sortAddressesByFractionDesc(
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) []string {

	// addressWithFraction pairs an address with its fractional remainder for sorting in the
	// Largest Remainder Method (LRM) distribution algorithm.
	type addressWithFraction struct {
		address  string
		fraction *big.Rat
	}
	var addressFractions []addressWithFraction

	// Calculate fractional remainders for each address
	for addrStr := range stakeAmounts {
		stake := stakeAmounts[addrStr]
		// Calculate exact reward with full precision
		exactReward := new(big.Rat).SetFrac(
			new(big.Int).Mul(stake.BigInt(), totalRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Extract fractional remainder
		baseReward := new(big.Int).Quo(exactReward.Num(), exactReward.Denom())
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactReward, baseRat)

		// Only include addresses with non-zero fractional parts
		if fractionalPart.Sign() > 0 {
			addressFractions = append(addressFractions, addressWithFraction{
				address:  addrStr,
				fraction: fractionalPart,
			})
		}
	}

	// Sort by fraction (descending), then by address (ascending) for determinism
	sort.Slice(addressFractions, func(i, j int) bool {
		cmp := addressFractions[i].fraction.Cmp(addressFractions[j].fraction)
		if cmp == 0 {
			// Tie-breaker: lexicographical address order
			return addressFractions[i].address < addressFractions[j].address
		}
		return cmp > 0 // Descending (largest fractions first)
	})

	// Extract sorted addresses
	var addressesWithFractions []string
	for _, af := range addressFractions {
		addressesWithFractions = append(addressesWithFractions, af.address)
	}

	return addressesWithFractions
}

// sortAddressesByStakeDesc sorts addresses by stake amount (descending).
// Uses lexicographical address ordering as tie-breaker for determinism.
// TODO_TEST: Add specific test case verifying deterministic ordering with equal stakes
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

	// Primary sort: stake descending; Secondary sort: address ascending (tie-breaker)
	sort.Slice(addressStakes, func(i, j int) bool {
		if addressStakes[i].stake.Equal(addressStakes[j].stake) {
			return addressStakes[i].address < addressStakes[j].address
		}
		return addressStakes[i].stake.GT(addressStakes[j].stake)
	})

	sortedAddresses := make([]string, len(addressStakes))
	for i, addrStake := range addressStakes {
		sortedAddresses[i] = addrStake.address
	}

	return sortedAddresses
}
