package types

import (
	"fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		MorseAccountState:     nil,
		MorseAccountClaimList: []MorseAccountClaim{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in morseAccountClaim
	morseAccountClaimIndexMap := make(map[string]struct{})

	for _, elem := range gs.MorseAccountClaimList {
		index := string(MorseAccountClaimKey(elem.MorseSrcAddress))
		if _, ok := morseAccountClaimIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for morseAccountClaim")
		}
		morseAccountClaimIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
