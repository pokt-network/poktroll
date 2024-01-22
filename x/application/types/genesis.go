package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
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

	// Check that the stake value for the apps is valid and that the delegatee addresses are valid
	for _, app := range gs.ApplicationList {
		// TODO_TECHDEBT: Consider creating shared helpers across the board for stake validation,
		// similar to how we have `ValidateAppServiceConfigs` below
		if app.Stake == nil {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "nil stake amount for application")
		}
		stake, err := sdk.ParseCoinNormalized(app.Stake.String())
		if !stake.IsValid() {
			return sdkerrors.Wrapf(
				ErrAppInvalidStake,
				"invalid stake amount for application %v; (%v)",
				app.Stake, stake.Validate(),
			)
		}
		if err != nil {
			return sdkerrors.Wrapf(
				ErrAppInvalidStake,
				"cannot parse stake amount for application %v; (%v)",
				app.Stake, err,
			)
		}
		if stake.IsZero() || stake.IsNegative() {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount for application: %v <= 0", app.Stake)
		}
		if stake.Denom != "upokt" {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount denom for application %v", app.Stake)
		}

		// Check that the application's delegated gateway addresses are valid
		for _, gatewayAddr := range app.DelegateeGatewayAddresses {
			if _, err := sdk.AccAddressFromBech32(gatewayAddr); err != nil {
				return sdkerrors.Wrapf(
					ErrAppInvalidGatewayAddress,
					"invalid gateway address %s; (%v)",
					gatewayAddr, err,
				)
			}
		}

		// Validate the application service configs
		if err := servicehelpers.ValidateAppServiceConfigs(app.ServiceConfigs); err != nil {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, err.Error())
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
