package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyComputeToTokensMultiplier = []byte("ComputeToTokensMultiplier")
	// TODO: Determine the default value
	DefaultComputeToTokensMultiplier uint64 = 0
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	computeToTokensMultiplier uint64,
) Params {
	return Params{
		ComputeToTokensMultiplier: computeToTokensMultiplier,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultComputeToTokensMultiplier,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyComputeToTokensMultiplier, &p.ComputeToTokensMultiplier, validateComputeToTokensMultiplier),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateComputeToTokensMultiplier(p.ComputeToTokensMultiplier); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateComputeToTokensMultiplier validates the ComputeToTokensMultiplier param
func validateComputeToTokensMultiplier(v interface{}) error {
	computeToTokensMultiplier, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = computeToTokensMultiplier

	return nil
}
