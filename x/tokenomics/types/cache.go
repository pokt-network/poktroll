package types

type Cache struct {
	Params *Params
}

func (c *Cache) Clear() {
	c.Params = nil
}
