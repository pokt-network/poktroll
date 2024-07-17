package application

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const ParamMaxDelegatedGateways = "max_delegated_gateways"

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyMaxDelegatedGateways = []byte("MaxDelegatedGateways")
	// TODO_MAINNET: Determine the default value
	DefaultMaxDelegatedGateways uint64 = 7
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(maxDelegatedGateways uint64) Params {
	return Params{
		MaxDelegatedGateways: maxDelegatedGateways,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMaxDelegatedGateways)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxDelegatedGateways, &p.MaxDelegatedGateways, validateMaxDelegatedGateways),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMaxDelegatedGateways(p.MaxDelegatedGateways); err != nil {
		return err
	}

	return nil
}

// validateMaxDelegatedGateways validates the MaxDelegatedGateways param
func validateMaxDelegatedGateways(v interface{}) error {
	maxDelegatedGateways, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// Hard-coding a value of 1 because we never expect this to change.
	// If an application choses to delegate, at least one is required.
	if maxDelegatedGateways < 1 {
		return ErrAppInvalidMaxDelegatedGateways.Wrapf("MaxDelegatedGateways param < 1: got %d", maxDelegatedGateways)
	}

	return nil
}
