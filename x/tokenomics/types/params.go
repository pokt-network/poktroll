package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyComputeUnitsToTokensMultiplier            = []byte("ComputeUnitsToTokensMultiplier")
	ParamComputeUnitsToTokensMultiplier          = "compute_units_to_tokens_multiplier"
	DefaultComputeUnitsToTokensMultiplier uint64 = 42 // TODO_MAINNET: Determine the default value.
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(computeUnitsToTokensMultiplier uint64) Params {
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
		paramtypes.NewParamSetPair(
			KeyComputeUnitsToTokensMultiplier,
			&p.ComputeUnitsToTokensMultiplier,
			ValidateComputeUnitsToTokensMultiplier,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	// Validate the ComputeUnitsToTokensMultiplier
	if err := ValidateComputeUnitsToTokensMultiplier(params.ComputeUnitsToTokensMultiplier); err != nil {
		return err
	}

	return nil
}

// ValidateComputeUnitsToTokensMultiplier validates the ComputeUnitsToTokensMultiplier governance parameter.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateComputeUnitsToTokensMultiplier(v interface{}) error {
	computeUnitsToTokensMultiplier, ok := v.(uint64)
	if !ok {
		return ErrTokenomicsParamsInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if computeUnitsToTokensMultiplier <= 0 {
		return ErrTokenomicsParamsInvalid.Wrapf("invalid ComputeUnitsToTokensMultiplier: (%v)", computeUnitsToTokensMultiplier)
	}

	return nil
}
