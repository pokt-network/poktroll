package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ServiceList:               []sharedtypes.Service{},
		RelayMiningDifficultyList: []RelayMiningDifficulty{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in services
	if err := validateServiceList(gs.ServiceList); err != nil {
		return err
	}

	// Check for duplicated index in relayMiningDifficulty
	if err := validateRelayMiningDifficultyList(gs.RelayMiningDifficultyList); err != nil {
		return err
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.ValidateBasic()
}

// validateServiceList validates the service list.
func validateServiceList(serviceList []sharedtypes.Service) error {
	serviceIDMap := make(map[string]struct{})
	serviceNameMap := make(map[string]struct{})
	for _, service := range serviceList {
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

	return nil
}

// validateRelayMiningDifficultyList validates the relayMiningDifficulty list.
func validateRelayMiningDifficultyList(relayMiningDifficultyList []RelayMiningDifficulty) error {
	relayMiningDifficultyIndexMap := make(map[string]struct{})

	for _, elem := range relayMiningDifficultyList {
		index := string(RelayMiningDifficultyKey(elem.ServiceId))
		if _, ok := relayMiningDifficultyIndexMap[index]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated index for relayMiningDifficulty: %s", index)
		}
		relayMiningDifficultyIndexMap[index] = struct{}{}
	}

	return nil
}
