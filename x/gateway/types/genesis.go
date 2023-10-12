package types

import (
	"fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		GatewayList: []Gateway{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in gateway
	gatewayIndexMap := make(map[string]struct{})

	for _, elem := range gs.GatewayList {
		index := string(GatewayKey(elem.Address))
		if _, ok := gatewayIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for gateway")
		}
		gatewayIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
