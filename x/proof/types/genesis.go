package types

import (
	"fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ClaimList: []Claim{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in claim
	claimIndexMap := make(map[string]struct{})

	// Ensure claims are unique with respect to a given session ID and supplier address.
	for _, claim := range gs.ClaimList {
		// TODO_BLOCKER: ensure the corresponding supplier exists and is staked.

		if claim.GetRootHash() == nil {
			return fmt.Errorf("root hash cannot be nil")
		}

		if len(claim.GetRootHash()) == 0 {
			return fmt.Errorf("root hash cannot be empty")
		}

		sessionId := claim.GetSessionHeader().GetSessionId()
		primaryKey := string(ClaimPrimaryKey(sessionId, claim.SupplierAddress))
		if _, ok := claimIndexMap[primaryKey]; ok {
			return fmt.Errorf("duplicated supplierAddr for claim")
		}
		claimIndexMap[primaryKey] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
