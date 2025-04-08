package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyWaiveMorseClaimGasFees     = []byte("WaiveMorseClaimGasFees")
	ParamWaiveMorseClaimGasFees   = "waive_morse_claim_gas_fees"
	DefaultWaiveMorseClaimGasFees = false
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	waiveMorseClaimGasFees bool,
) Params {
	return Params{
		WaiveMorseClaimGasFees: waiveMorseClaimGasFees,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultWaiveMorseClaimGasFees,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyWaiveMorseClaimGasFees,
			&p.WaiveMorseClaimGasFees,
			ValidateWaiveMorseClaimGasFees,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := ValidateWaiveMorseClaimGasFees(p.WaiveMorseClaimGasFees); err != nil {
		return err
	}
	return nil
}

// ValidateWaiveMorseClaimGasFees validates the WaiveMorseClaimGasFees param.
func ValidateWaiveMorseClaimGasFees(waiveMorseClaimGasFeesAny any) error {
	if _, ok := waiveMorseClaimGasFeesAny.(bool); !ok {
		return ErrMigrationParamInvalid.Wrapf("invalid parameter type: %T", waiveMorseClaimGasFeesAny)
	}
	return nil
}
