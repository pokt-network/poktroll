package types

type Cache struct {
	Params *Params
	Claims map[string]*Claim
	Proofs map[string]*Proof
}

func (c *Cache) Clear() {
	c.Params = nil
	clear(c.Claims)
	clear(c.Proofs)
}
