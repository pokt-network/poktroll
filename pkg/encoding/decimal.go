package encoding

import (
	"fmt"
	"math/big"
	"strconv"
)

// Float64ToRat converts a float64 to a big.Rat for precise decimal arithmetic.
// TODO_FUTURE: Consider switching to string representations for tokenomics % allocations
// since CosmosSDK will deprecate float64 values with zero copy encoding of scalar values.
// Ref: https://docs.cosmos.network/main/build/rfc/rfc-002-zero-copy-encoding
// NB: It is publicly exposed to be used in the tests.
func Float64ToRat(f float64) (*big.Rat, error) {
	// Convert float64 to string before big.Rat conversion to avoid floating point precision issues
	// Example:
	// - bigRat.SetString("0.1") == 1/10
	// - bigRat.SetFloat64(0.1) == 3602879701896397/36028797018963968
	formattedFloat := strconv.FormatFloat(f, 'f', -1, 64)
	ratio, ok := new(big.Rat).SetString(formattedFloat)
	if !ok {
		return nil, fmt.Errorf("error converting float64 to big.Rat: %f", f)
	}

	return ratio, nil
}
