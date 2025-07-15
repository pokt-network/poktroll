package types

import "math"

const basisPointsTotal = 10000 // 100% = 10000 basis points

// Sum returns the sum of all mint allocation percentages.
// It uses basis points internally for exact arithmetic validation.
// TODO_IMPROVE(@red-0ne): Use string % representations and big.Rat in the future.
func (m *MintAllocationPercentages) Sum() float64 {
	// Convert each percentage to basis points for exact integer arithmetic
	daoBP := int64(math.Round(m.Dao * float64(basisPointsTotal)))
	proposerBP := int64(math.Round(m.Proposer * float64(basisPointsTotal)))
	supplierBP := int64(math.Round(m.Supplier * float64(basisPointsTotal)))
	sourceOwnerBP := int64(math.Round(m.SourceOwner * float64(basisPointsTotal)))
	applicationBP := int64(math.Round(m.Application * float64(basisPointsTotal)))

	// Sum basis points
	sumBP := daoBP + proposerBP + supplierBP + sourceOwnerBP + applicationBP

	// Convert back to percentage
	return float64(sumBP) / float64(basisPointsTotal)
}
