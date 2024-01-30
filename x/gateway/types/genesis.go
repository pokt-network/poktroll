package types

import (
	"fmt"
)

// DefaultAddress is the default global index
const DefaultAddress uint64 = 1

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
	gatewayAddressMap := make(map[string]struct{})

	for _, gateway := range gs.GatewayList {
		addr := string(GatewayKey(gateway.Address))
		if _, ok := gatewayAddressMap[addr]; ok {
			return fmt.Errorf("duplicated address for gateway")
		}
		gatewayAddressMap[addr] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
