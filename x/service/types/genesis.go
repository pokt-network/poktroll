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

	if err := validateSupplierAllowlist(gs.ServiceList, gs.SupplierAllowlist, gs.SupplierAllowlistSweeps); err != nil {
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
		if err := service.ValidateBasic(); err != nil {
			return err
		}

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

// validateSupplierAllowlist validates the supplier allowlist entries and pending
// sweeps against the (already validated) service list.
func validateSupplierAllowlist(
	serviceList []sharedtypes.Service,
	entries []SupplierAllowlistEntry,
	sweeps []AllowlistSweep,
) error {
	serviceIds := make(map[string]struct{}, len(serviceList))
	for _, service := range serviceList {
		serviceIds[service.Id] = struct{}{}
	}

	entryKeys := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := serviceIds[entry.ServiceId]; !ok {
			return ErrServiceInvalidSupplierAllowlist.Wrapf("allowlist entry of unknown service %q", entry.ServiceId)
		}
		if err := sharedtypes.ValidateCanonicalAddress(entry.Address); err != nil {
			return ErrServiceInvalidSupplierAllowlist.Wrapf(
				"invalid allowlist address %q of service %q; (%v)", entry.Address, entry.ServiceId, err,
			)
		}
		entryKey := string(SupplierAllowlistKey(entry.ServiceId, entry.Address))
		if _, ok := entryKeys[entryKey]; ok {
			return ErrServiceDuplicateIndex.Wrapf(
				"duplicated allowlist entry %q of service %q", entry.Address, entry.ServiceId,
			)
		}
		entryKeys[entryKey] = struct{}{}
	}

	sweepServiceIds := make(map[string]struct{}, len(sweeps))
	for _, sweep := range sweeps {
		if _, ok := serviceIds[sweep.ServiceId]; !ok {
			return ErrServiceInvalidSupplierAllowlist.Wrapf("allowlist sweep of unknown service %q", sweep.ServiceId)
		}
		if _, ok := sweepServiceIds[sweep.ServiceId]; ok {
			return ErrServiceDuplicateIndex.Wrapf("duplicated allowlist sweep of service %q", sweep.ServiceId)
		}
		sweepServiceIds[sweep.ServiceId] = struct{}{}
		if sweep.DeferredFromHeight < 0 {
			return ErrServiceInvalidSupplierAllowlist.Wrapf(
				"negative deferred_from_height %d of the allowlist sweep of service %q",
				sweep.DeferredFromHeight, sweep.ServiceId,
			)
		}
		// A deferred sweep has not started, so it has no cursor: the first pass
		// drops deferred_from_height when it writes one.
		if sweep.DeferredFromHeight != 0 && len(sweep.Cursor) > 0 {
			return ErrServiceInvalidSupplierAllowlist.Wrapf(
				"deferred allowlist sweep of service %q carries a cursor", sweep.ServiceId,
			)
		}
	}

	return nil
}
