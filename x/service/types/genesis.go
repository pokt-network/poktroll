package types

import (
	"fmt"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ServiceList: []sharedtypes.Service{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in service
	serviceIndexMap := make(map[string]struct{})

	for _, elem := range gs.ServiceList {
		id := string(ServiceKey(elem.Id))
		if _, ok := serviceIndexMap[id]; ok {
			return fmt.Errorf("duplicated id for service")
		}
		serviceIndexMap[id] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
