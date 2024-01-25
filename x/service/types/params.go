package types

import (
	sdkerrors "cosmossdk.io/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

// DefaultAddServiceFee is the default value for the add service fee
// parameter in the genesis state of the service module.
// TODO_BLOCKER: Revisit default param values for service fee
const DefaultAddServiceFee = 1000000000 // 1000 POKT

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{AddServiceFee: DefaultAddServiceFee}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	// TODO(@h5law): Look into better validation
	if p.AddServiceFee < DefaultAddServiceFee {
		return sdkerrors.Wrapf(
			ErrServiceInvalidServiceFee,
			"AddServiceFee param %d uPOKT: got %d",
			DefaultAddServiceFee,
			p.AddServiceFee,
		)
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
