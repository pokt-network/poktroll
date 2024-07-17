package tokenomics

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
		index := elem.ServiceId //string(RelayMiningDifficultyKey(elem.ServiceId))
		if _, ok := relayMiningDifficultyIndexMap[index]; ok {
			return ErrTokenomicsDuplicateIndex.Wrapf("duplicated index for relayMiningDifficulty: %s", index)
		}
		relayMiningDifficultyIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.ValidateBasic()
}
