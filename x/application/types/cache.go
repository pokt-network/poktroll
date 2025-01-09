package types

type Cache struct {
	Params       *Params
	Applications map[string]*Application
}

func (c *Cache) Clear() {
	c.Params = nil
	clear(c.Applications)
}
