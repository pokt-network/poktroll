package types

import (
	"fmt"

	sharedtypes "pocket/x/shared/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		SupplierList: []sharedtypes.Supplier{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in supplier
	supplierIndexMap := make(map[string]struct{})

	for _, supplier := range gs.SupplierList {
		index := string(SupplierKey(supplier.Address))
		if _, ok := supplierIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for supplier")
		}
		supplierIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
