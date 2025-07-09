package types

// Sum returns the sum of all mint equals burn claim distribution percentages.
func (c *MintEqualsBurnClaimDistribution) Sum() float64 {
	return c.Dao + c.Proposer + c.Supplier + c.SourceOwner + c.Application
}
