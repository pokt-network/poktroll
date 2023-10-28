package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "pocket/x/shared/helpers"
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

	// Check that the stake value for the apps is valid and that the delegatee pubkeys are valid
	for _, app := range gs.ApplicationList {
		// TODO_TECHDEBT: Consider creating shared helpers across the board for stake validation,
		// similar to how we have `AreValidAppServiceConfigs` below
		if app.Stake == nil {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "nil stake amount for application")
		}
		stake, err := sdk.ParseCoinNormalized(app.Stake.String())
		if !stake.IsValid() {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount for application %v; (%v)", app.Stake, stake.Validate())
		}
		if err != nil {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "cannot parse stake amount for application %v; (%v)", app.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount for application: %v <= 0", app.Stake)
		}
		if stake.Denom != "upokt" {
			return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount denom for application %v", app.Stake)
		}

		// Check that the application's delegated gateway pubkeys are valid
		for _, gatewayPubKey := range app.DelegateeGatewayPubKeys {
			if _, err := AnyToPubKey(gatewayPubKey); err != nil {
				return sdkerrors.Wrapf(ErrAppAnyIsNotPubKey, "invalid delegatee pubkey for application %v", gatewayPubKey)
			}
		}

		// Validate the application service configs
		if reason, ok := servicehelpers.AreValidAppServiceConfigs(app.ServiceConfigs); !ok {
			return sdkerrors.Wrapf(ErrAppInvalidStake, reason)
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
