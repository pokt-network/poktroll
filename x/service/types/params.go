package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyAddServiceFee = []byte("AddServiceFee")
	// TODO: Determine the default value
	DefaultAddServiceFee uint64 = 0
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	addServiceFee uint64,
) Params {
	return Params{
		AddServiceFee: addServiceFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultAddServiceFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAddServiceFee, &p.AddServiceFee, validateAddServiceFee),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateAddServiceFee(p.AddServiceFee); err != nil {
		return err
	}

	return nil
}

// validateAddServiceFee validates the AddServiceFee param
func validateAddServiceFee(v interface{}) error {
	addServiceFee, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = addServiceFee

	return nil
}
