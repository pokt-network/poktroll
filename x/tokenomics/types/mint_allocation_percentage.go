package types

// Sum returns the sum of all mint allocation percentages.
func (m *MintAllocationPercentages) Sum() float64 {
	return m.Dao + m.Proposer + m.Supplier + m.SourceOwner + m.Application
}
