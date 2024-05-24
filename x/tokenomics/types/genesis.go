package types

import (
"fmt"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		RelayMiningDifficultyList: []RelayMiningDifficulty{},
// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in relayMiningDifficulty
relayMiningDifficultyIndexMap := make(map[string]struct{})

for _, elem := range gs.RelayMiningDifficultyList {
	index := string(RelayMiningDifficultyKey(elem.ServiceId))
	if _, ok := relayMiningDifficultyIndexMap[index]; ok {
		return fmt.Errorf("duplicated index for relayMiningDifficulty")
	}
	relayMiningDifficultyIndexMap[index] = struct{}{}
}
// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.ValidateBasic()
}
