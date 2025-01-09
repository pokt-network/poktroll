package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

type Cache struct {
	Params                *Params
	Services              map[string]*sharedtypes.Service
	RelayMiningDifficulty map[string]*RelayMiningDifficulty
}

func (c *Cache) Clear() {
	c.Params = nil
	clear(c.Services)
	clear(c.RelayMiningDifficulty)
}
