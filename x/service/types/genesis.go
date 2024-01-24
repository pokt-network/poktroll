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
	serviceIDIndexMap := make(map[string]struct{})
	serviceNameIndexMap := make(map[string]struct{})
	for _, service := range gs.ServiceList {
		idIndex := string(ServiceKey(service.Id))
		if _, ok := serviceIDIndexMap[idIndex]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated ID for service: %v", service)
		}
		serviceIDIndexMap[idIndex] = struct{}{}
		nameIndex := string(ServiceKey(service.Name))
		if _, ok := serviceNameIndexMap[nameIndex]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated name for service: %v", service)
		}
		serviceNameIndexMap[nameIndex] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
