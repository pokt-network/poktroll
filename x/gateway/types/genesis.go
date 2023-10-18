package types

import (
	sdkerrors "cosmossdk.io/errors"
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

	for _, gateway := range gs.GatewayList {
		// Check for duplicated index in gateway
		index := string(GatewayKey(gateway.Address))
		if _, ok := gatewayIndexMap[index]; ok {
			return sdkerrors.Wrap(ErrGatewayInvalidAddress, "duplicated index for gateway")
		}
		gatewayIndexMap[index] = struct{}{}
		// Validate the stake of each gateway
		if gateway.Stake == nil {
			return sdkerrors.Wrap(ErrGatewayInvalidStake, "nil stake amount for gateway")
		}
		stake, err := sdk.ParseCoinNormalized(gateway.Stake.String())
		if !stake.IsValid() {
			return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount for gateway %v; (%v)", gateway.Stake, stake.Validate())
		}
		if err != nil {
			return sdkerrors.Wrapf(ErrGatewayInvalidStake, "cannot parse stake amount for gateway %v; (%v)", gateway.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount for gateway: %v <= 0", gateway.Stake)
		}
		if stake.Denom != "upokt" {
			return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount denom for gateway %v", gateway.Stake)
		}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
