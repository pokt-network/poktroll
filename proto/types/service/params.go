package service

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultAddServiceFee is the default value for the add service fee
// parameter in the genesis state of the service module.
// TODO_BETA: Revisit default param values for service fee
const (
	ParamAddServiceFee   = "add_service_fee"
	DefaultAddServiceFee = 1000000000 // 1000 POKT
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyAddServiceFee = []byte("AddServiceFee")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(addServiceFee uint64) Params {
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
	// TODO_MAINNET: Look into better validation
	if p.AddServiceFee < DefaultAddServiceFee {
		return ErrServiceInvalidServiceFee.Wrapf(
			"AddServiceFee param %d uPOKT: got %d",
			DefaultAddServiceFee,
			p.AddServiceFee,
		)
	}
	return nil
}

// validateAddServiceFee validates the AddServiceFee param
func validateAddServiceFee(v interface{}) error {
	addServiceFee, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO_BETA: implement validation
	_ = addServiceFee

	return nil
}
