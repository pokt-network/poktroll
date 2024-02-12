package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyComputeUnitsToTokensMultiplier = []byte("ComputeUnitsToTokensMultiplier")
	// TODO: Determine the default value
	DefaultComputeUnitsToTokensMultiplier uint64 = 42
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	computeUnitsToTokensMultiplier uint64,
) Params {
	return Params{
		ComputeUnitsToTokensMultiplier: computeUnitsToTokensMultiplier,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultComputeUnitsToTokensMultiplier,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyComputeUnitsToTokensMultiplier, &p.ComputeUnitsToTokensMultiplier, validateComputeUnitsToTokensMultiplier),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateComputeUnitsToTokensMultiplier(p.ComputeUnitsToTokensMultiplier); err != nil {
		return err
	}

	return nil
}

// validateComputeUnitsToTokensMultiplier validates the ComputeUnitsToTokensMultiplier param
func validateComputeUnitsToTokensMultiplier(v interface{}) error {
	computeUnitsToTokensMultiplier, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if computeUnitsToTokensMultiplier <= 0 {
		return fmt.Errorf("invalid compute to tokens multiplier: %d", computeUnitsToTokensMultiplier)
	}

	return nil
}
