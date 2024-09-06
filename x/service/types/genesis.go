package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

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
	serviceIDMap := make(map[string]struct{})
	serviceNameMap := make(map[string]struct{})
	for _, service := range gs.ServiceList {
		serviceID := string(ServiceKey(service.Id))
		if _, ok := serviceIDMap[serviceID]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated ID for service: %v", service)
		}
		serviceIDMap[serviceID] = struct{}{}
		serviceName := string(ServiceKey(service.Name))
		if _, ok := serviceNameMap[serviceName]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated name for service: %v", service)
		}
		serviceNameMap[serviceName] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.ValidateBasic()
}
