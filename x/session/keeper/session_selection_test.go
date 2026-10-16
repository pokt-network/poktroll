package keeper

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// referenceSelectCandidates is the supplier selection used before
// selectLowestWeightCandidates, kept as the oracle for the differential tests:
// sort every candidate by (weight, operator address) and keep the first n.
func referenceSelectCandidates(candidates []weightedCandidate, n int) []*sharedtypes.ServiceConfigUpdate {
	candidateSuppliers := make([]*sharedtypes.ServiceConfigUpdate, len(candidates))
	candidatesToRandomWeight := make(map[string]int, len(candidates))
	for i, candidate := range candidates {
		candidateSuppliers[i] = candidate.config
		candidatesToRandomWeight[candidate.config.OperatorAddress] = candidate.weight
	}

	weightedSupplierSortFn := func(serviceConfigUpdateA, serviceConfigUpdateB *sharedtypes.ServiceConfigUpdate) int {
		weightA := candidatesToRandomWeight[serviceConfigUpdateA.OperatorAddress]
		weightB := candidatesToRandomWeight[serviceConfigUpdateB.OperatorAddress]
		if weightA < weightB {
			return -1
		} else if weightA > weightB {
			return 1
		}
		return bytes.Compare(
			[]byte(serviceConfigUpdateA.OperatorAddress),
			[]byte(serviceConfigUpdateB.OperatorAddress),
		)
	}

	slices.SortFunc(candidateSuppliers, weightedSupplierSortFn)
	return candidateSuppliers[:n]
}

type weightMode int

const (
	weightsSessionHash weightMode = iota // production weights
	weightsFewValues                     // many equal weights: the address decides
	weightsExtremes                      // int64 bounds, where subtraction would overflow
)

// randomCandidates returns count candidates with distinct operator addresses.
// Addresses share prefixes so that the address tie-break compares deep bytes.
func randomCandidates(rng *rand.Rand, count int, mode weightMode, sessionIDBz []byte) []weightedCandidate {
	extremes := []int{math.MinInt64, math.MinInt64 + 1, -1, 0, 1, math.MaxInt64 - 1, math.MaxInt64}
	seen := make(map[string]struct{}, count)
	candidates := make([]weightedCandidate, 0, count)
	for len(candidates) < count {
		address := fmt.Sprintf("pokt1%03x%x", rng.IntN(16), rng.Uint64()%4096)
		if _, dup := seen[address]; dup {
			continue
		}
		seen[address] = struct{}{}

		var weight int
		switch mode {
		case weightsSessionHash:
			weight = generateSupplierRandomWeight(address, sessionIDBz)
		case weightsFewValues:
			weight = rng.IntN(4) - 2
		case weightsExtremes:
			weight = extremes[rng.IntN(len(extremes))]
		}
		candidates = append(candidates, weightedCandidate{
			weight: weight,
			config: &sharedtypes.ServiceConfigUpdate{OperatorAddress: address},
		})
	}
	return candidates
}

func operatorAddresses(configs []*sharedtypes.ServiceConfigUpdate) []string {
	addresses := make([]string, len(configs))
	for i, config := range configs {
		addresses[i] = config.OperatorAddress
	}
	return addresses
}

// TestSelectLowestWeightCandidates_MatchesFullSort checks that the bounded
// selection returns exactly the suppliers, in exactly the order, that the full
// sort returns, for any input order. It covers the hydrator's range of inputs:
// candidate count >= n (fewer candidates bypass the selection).
func TestSelectLowestWeightCandidates_MatchesFullSort(t *testing.T) {
	rng := rand.New(rand.NewPCG(20261016, 1))

	for trial := 0; trial < 3000; trial++ {
		mode := weightMode(trial % 3)
		n := rng.IntN(60)
		count := n + rng.IntN(150)
		if trial%100 == 0 {
			count = n + 4032 // mainnet-sized candidate list
		}
		sessionIDBz := make([]byte, 32)
		for i := range sessionIDBz {
			sessionIDBz[i] = byte(rng.Uint32())
		}
		candidates := randomCandidates(rng, count, mode, sessionIDBz)

		expected := operatorAddresses(referenceSelectCandidates(candidates, n))
		require.Equal(t, expected, operatorAddresses(selectLowestWeightCandidates(candidates, n)),
			"trial %d (mode %d, n %d, count %d)", trial, mode, n, count)

		shuffled := slices.Clone(candidates)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		require.Equal(t, expected, operatorAddresses(selectLowestWeightCandidates(shuffled, n)),
			"trial %d shuffled (mode %d, n %d, count %d)", trial, mode, n, count)
	}
}

// TestSelectLowestWeightCandidates_KeepsConfigs checks that the selection
// returns the candidates' own config pointers, which the hydrator memoizes and
// resolves to suppliers.
func TestSelectLowestWeightCandidates_KeepsConfigs(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 7))
	candidates := randomCandidates(rng, 200, weightsSessionHash, []byte("session"))

	selected := selectLowestWeightCandidates(candidates, 15)
	require.Len(t, selected, 15)
	for _, config := range selected {
		require.True(t, slices.ContainsFunc(candidates, func(c weightedCandidate) bool { return c.config == config }))
	}
}
