package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	// WaiveMorseClaimGasFees	
	KeyWaiveMorseClaimGasFees               = []byte("WaiveMorseClaimGasFees")
	ParamWaiveMorseClaimGasFees             = "waive_morse_claim_gas_fees"
	DefaultWaiveMorseClaimGasFees           = false

	// AllowMorseAccountImportOverwrite
	KeyAllowMorseAccountImportOverwrite     = []byte("AllowMorseAccountImportOverwrite")
	ParamAllowMorseAccountImportOverwrite   = "allow_morse_account_import_overwrite"
	DefaultAllowMorseAccountImportOverwrite = false
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	waiveMorseClaimGasFees bool,
	allowMorseAccountImportOverwrite bool,
) Params {
	return Params{
		WaiveMorseClaimGasFees:           waiveMorseClaimGasFees,
		AllowMorseAccountImportOverwrite: allowMorseAccountImportOverwrite,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultWaiveMorseClaimGasFees,
		DefaultAllowMorseAccountImportOverwrite,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyWaiveMorseClaimGasFees,
			&p.WaiveMorseClaimGasFees,
			ValidateParamIsBool,
		),
		paramtypes.NewParamSetPair(
			KeyAllowMorseAccountImportOverwrite,
			&p.AllowMorseAccountImportOverwrite,
			ValidateParamIsBool,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := ValidateParamIsBool(p.WaiveMorseClaimGasFees); err != nil {
		return err
	}

	if err := ValidateParamIsBool(p.AllowMorseAccountImportOverwrite); err != nil {
		return err
	}

	return nil
}

// ValidateParamIsBool validates that the param is a boolean type.
func ValidateParamIsBool(paramAny any) error {
	if _, ok := paramAny.(bool); !ok {
		return ErrMigrationParamInvalid.Wrapf("invalid parameter type: %T", paramAny)
	}
	return nil
}
