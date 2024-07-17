package application

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

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
	applicationAddrMap := make(map[string]struct{})

	for _, app := range gs.ApplicationList {
		appAddr := app.Address //string(ApplicationKey(app.Address))
		if _, ok := applicationAddrMap[appAddr]; ok {
			return fmt.Errorf("duplicated index for application")
		}
		applicationAddrMap[appAddr] = struct{}{}
	}

	// Check that the stake value for the apps is valid and that the delegatee addresses are valid
	for _, app := range gs.ApplicationList {
		// TODO_MAINNET: Consider creating shared helpers across the board for stake validation,
		// similar to how we have `ValidateAppServiceConfigs` below
		if app.Stake == nil {
			return ErrAppInvalidStake.Wrapf("nil stake amount for application")
		}
		stake, err := sdk.ParseCoinNormalized(app.Stake.String())
		if !stake.IsValid() {
			return ErrAppInvalidStake.Wrapf("invalid stake amount for application %v; (%v)", app.Stake, stake.Validate())
		}
		if err != nil {
			return ErrAppInvalidStake.Wrapf("cannot parse stake amount for application %v; (%v)", app.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return ErrAppInvalidStake.Wrapf("invalid stake amount for application: %v <= 0", app.Stake)
		}
		if stake.Denom != "upokt" {
			return ErrAppInvalidStake.Wrapf("invalid stake amount denom for application %v", app.Stake)
		}

		// Check that the application's delegated gateway addresses are valid
		for _, gatewayAddr := range app.DelegateeGatewayAddresses {
			if _, err := sdk.AccAddressFromBech32(gatewayAddr); err != nil {
				return ErrAppInvalidGatewayAddress.Wrapf("invalid gateway address %s; (%v)", gatewayAddr, err)
			}
		}

		// Validate the application service configs
		if err := servicehelpers.ValidateAppServiceConfigs(app.ServiceConfigs); err != nil {
			return ErrAppInvalidServiceConfigs.Wrapf(err.Error())
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
