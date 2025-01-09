package types

type Cache struct {
	BlockHashes map[int64][]byte
	Params      *Params
}

func (c *Cache) Clear() {
	c.Params = nil
	clear(c.BlockHashes)
}
