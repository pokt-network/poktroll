package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		GatewayList: []Gateway{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	gatewayIndexMap := make(map[string]struct{})

	for _, elem := range gs.GatewayList {
		// Check for duplicated index in gateway
		index := string(GatewayKey(elem.Address))
		if _, ok := gatewayIndexMap[index]; ok {
			return errors.Wrap(ErrGatewayInvalidAddress, "duplicated index for gateway")
		}
		gatewayIndexMap[index] = struct{}{}
		// Validate the stake of each gateway
		if elem.Stake == nil {
			return errors.Wrap(ErrGatewayInvalidStake, "nil stake amount for gateway")
		}
		stakeAmount, err := sdk.ParseCoinNormalized(elem.Stake.String())
		if !stakeAmount.IsValid() {
			return errors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount for gateway %v; (%v)", elem.Stake, stakeAmount.Validate())
		}
		if err != nil {
			return errors.Wrapf(ErrGatewayInvalidStake, "cannot parse stake amount for gateway %v; (%v)", elem.Stake, err)
		}
		if stakeAmount.IsZero() || stakeAmount.IsNegative() {
			return errors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount for gateway: %v <= 0", elem.Stake)
		}
		if stakeAmount.Denom != "upokt" {
			return errors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount denom for gateway %v", elem.Stake)
		}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
