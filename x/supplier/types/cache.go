package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

type Cache struct {
	Params    *Params
	Suppliers map[string]*sharedtypes.Supplier
}

func (c *Cache) Clear() {
	c.Params = nil
	clear(c.Suppliers)
}
