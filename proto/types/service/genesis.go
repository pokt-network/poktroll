package service

import "github.com/pokt-network/poktroll/proto/types/shared"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ServiceList: []shared.Service{},
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
		serviceID := service.Id //string(ServiceKey(service.Id))
		if _, ok := serviceIDMap[serviceID]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated ID for service: %v", service)
		}
		serviceIDMap[serviceID] = struct{}{}
		serviceName := service.Name //string(ServiceKey(service.Name))
		if _, ok := serviceNameMap[serviceName]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated name for service: %v", service)
		}
		serviceNameMap[serviceName] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
