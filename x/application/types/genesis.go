package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ApplicationList: []Application{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in application
	applicationIndexMap := make(map[string]struct{})
	for _, elem := range gs.ApplicationList {
		index := string(ApplicationKey(elem.Address))
		if _, ok := applicationIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for application")
		}
		applicationIndexMap[index] = struct{}{}
	}

	// Check that the stake value for the apps is valid
	for _, elem := range gs.ApplicationList {
		if elem.Stake == nil {
			return fmt.Errorf("nil stake amount for application")
		}
		stakeAmount, err := sdk.ParseCoinNormalized(elem.Stake.String())
		if !stakeAmount.IsValid() {
			return fmt.Errorf("invalid stake amount for application %v; (%v)", elem.Stake, stakeAmount.Validate())
		}
		if err != nil {
			return fmt.Errorf("cannot parse stake amount for application %v; (%v)", elem.Stake, err)
		}
		if stakeAmount.IsZero() || stakeAmount.IsNegative() {
			return fmt.Errorf("invalid stake amount for application: %v <= 0", elem.Stake)
		}
		if stakeAmount.Denom != "upokt" {
			return fmt.Errorf("invalid stake amount denom for application %v", elem.Stake)
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
