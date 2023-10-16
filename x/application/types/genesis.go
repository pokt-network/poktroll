package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
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
	for _, app := range gs.ApplicationList {
		index := string(ApplicationKey(app.Address))
		if _, ok := applicationIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for application")
		}
		applicationIndexMap[index] = struct{}{}
	}

	// Check that the stake value for the apps is valid
	for _, app := range gs.ApplicationList {
		if app.Stake == nil {
			return errorsmod.Wrapf(ErrAppInvalidStake, "nil stake amount for application")
		}
		stake, err := sdk.ParseCoinNormalized(app.Stake.String())
		if !stake.IsValid() {
			return errorsmod.Wrapf(ErrAppInvalidStake, "invalid stake amount for application %v; (%v)", app.Stake, stake.Validate())
		}
		if err != nil {
			return errorsmod.Wrapf(ErrAppInvalidStake, "cannot parse stake amount for application %v; (%v)", app.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return errorsmod.Wrapf(ErrAppInvalidStake, "invalid stake amount for application: %v <= 0", app.Stake)
		}
		if stake.Denom != "upokt" {
			return errorsmod.Wrapf(ErrAppInvalidStake, "invalid stake amount denom for application %v", app.Stake)
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
