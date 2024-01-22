package types

import (
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
	// Check for duplicated index in services
	serviceIndexMap := make(map[string]struct{})
	for _, service := range gs.ServiceList {
		index := string(ServiceKey(service.Id))
		if _, ok := serviceIndexMap[index]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated index for service: %s", index)
		}
		serviceIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
