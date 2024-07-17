package gateway

import sdk "github.com/cosmos/cosmos-sdk/types"

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
	gatewayAddrMap := make(map[string]struct{})

	for _, gateway := range gs.GatewayList {
		// Check for duplicated address in gateway
		address := gateway.Address //string(GatewayKey(gateway.Address))
		if _, ok := gatewayAddrMap[address]; ok {
			return ErrGatewayInvalidAddress.Wrap("duplicated index for gateway")
		}
		gatewayAddrMap[address] = struct{}{}
		// Validate the stake of each gateway
		if gateway.Stake == nil {
			return ErrGatewayInvalidStake.Wrap("nil stake amount for gateway")
		}
		stake, err := sdk.ParseCoinNormalized(gateway.Stake.String())
		if !stake.IsValid() {
			return ErrGatewayInvalidStake.Wrapf("invalid stake amount for gateway %v; (%v)", gateway.Stake, stake.Validate())
		}
		if err != nil {
			return ErrGatewayInvalidStake.Wrapf("cannot parse stake amount for gateway %v; (%v)", gateway.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return ErrGatewayInvalidStake.Wrapf("invalid stake amount for gateway: %v <= 0", gateway.Stake)
		}
		if stake.Denom != "upokt" {
			return ErrGatewayInvalidStake.Wrapf("invalid stake amount denom for gateway %v", gateway.Stake)
		}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
