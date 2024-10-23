package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DaoRewardAddress is the address that will receive the dao/foundation rewards
// during claim settlement (global mint TLM). It is intended to be assigned via
// -ldflags="-X github.com/pokt-network/poktroll/app.FoundationRewardAddress=<foundation_address>"
// at built-time. This can be done via the config.yaml by adding the line as an element
// to the build.ldflags list.
//
// TODO_TECHDEBT: Promote this value to a tokenomics module parameter.
var DaoRewardAddress string

var (
	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	return nil
}
