package token_logic_modules

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPermute checks the correctness of the permute function such that it is
// "safe" to use in tests (i.e. it does what it's supposed to).
func TestPermute(t *testing.T) {
	input := []string{"1", "2", "3", "4"}
	expected := map[string][]string{
		"1234": {"1", "2", "3", "4"},
		"1243": {"1", "2", "4", "3"},
		"1342": {"1", "3", "4", "2"},
		"1324": {"1", "3", "2", "4"},
		"1423": {"1", "4", "2", "3"},
		"1432": {"1", "4", "3", "2"},
		"2341": {"2", "3", "4", "1"},
		"2314": {"2", "3", "1", "4"},
		"2413": {"2", "4", "1", "3"},
		"2431": {"2", "4", "3", "1"},
		"2143": {"2", "1", "4", "3"},
		"2134": {"2", "1", "3", "4"},
		"3412": {"3", "4", "1", "2"},
		"3421": {"3", "4", "2", "1"},
		"3124": {"3", "1", "2", "4"},
		"3142": {"3", "1", "4", "2"},
		"3241": {"3", "2", "4", "1"},
		"3214": {"3", "2", "1", "4"},
		"4123": {"4", "1", "2", "3"},
		"4132": {"4", "1", "3", "2"},
		"4231": {"4", "2", "3", "1"},
		"4213": {"4", "2", "1", "3"},
		"4312": {"4", "3", "1", "2"},
		"4321": {"4", "3", "2", "1"},
	}

	actual := permute(t, input)
	require.Equal(t, factorial(len(input)), len(actual))

	// Assert that each actual result matches exactly one expected permutation.
	for _, actualPermutation := range expected {
		actualKey := strings.Join(actualPermutation, "")
		expectedPermutation, isExpectedPermutation := expected[actualKey]
		require.True(t, isExpectedPermutation)
		require.Equal(t, expectedPermutation, actualPermutation)

		// Remove observed expected permutation to identify any
		// missing permutations after the loop.
		delete(expected, actualKey)
	}
	// Assert that all expected permutations were observed (and deleted).
	require.Len(t, expected, 0)
}

// permute generates all possible permutations of the input slice 'items'.
// It is used to generate all possible permutations of token logic module
// orderings such that we can test for commutativity.
func permute[T any](t *testing.T, items []T) [][]T {
	t.Helper()

	var permutations [][]T
	// Create a copy to avoid modifying the original slice.
	itemsCopy := make([]T, len(items))
	copy(itemsCopy, items)
	// Start the recursive permutation generation with swap index 0.
	recursivePermute(t, itemsCopy, &permutations, 0)
	return permutations
}

// recursivePermute recursively generates permutations by swapping elements.
func recursivePermute[T any](t *testing.T, items []T, permutations *[][]T, swapIdx int) {
	t.Helper()

	if swapIdx == len(items) {
		// Append a copy of the current permutation to the result.
		permutation := make([]T, len(items))
		copy(permutation, items)
		*permutations = append(*permutations, permutation)
		return
	}
	for i := swapIdx; i < len(items); i++ {
		// Swap the current element with the element at the swap index.
		items[swapIdx], items[i] = items[i], items[swapIdx]
		// Recurse with the next swap index.
		recursivePermute[T](t, items, permutations, swapIdx+1)
		// Swap back to restore the original state (backtrack).
		items[swapIdx], items[i] = items[i], items[swapIdx]
	}
}

// factorial calculates the factorial of n (i.e. n! = 1 * 2 * 3 * ... * n).
// It is used to calculate the number of permutations for a given set of items in permute().
func factorial(n int) int {
	if n < 0 {
		return 0 // Handle negative input as an invalid case
	}
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}
