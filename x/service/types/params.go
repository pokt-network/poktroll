package types

import (
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/pokt-network/poktroll/app/volatile"
)

// DefaultAddServiceFee is the default value for the add service fee
// parameter in the genesis state of the service module.
// TODO_BETA: Revisit default param values for service fee
var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyAddServiceFee   = []byte("AddServiceFee")
	ParamAddServiceFee = "add_service_fee"
	// TODO_TECHDEBT: Determine a sensible default value for the add service fee.
	DefaultAddServiceFee = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000000000))
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(addServiceFee *cosmostypes.Coin) Params {
	return Params{
		AddServiceFee: addServiceFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		&DefaultAddServiceFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAddServiceFee, &p.AddServiceFee, ValidateAddServiceFee),
	}
}

// ValidateBasic validates the set of params
func (p Params) ValidateBasic() error {
	if err := ValidateAddServiceFee(p.AddServiceFee); err != nil {
		return err
	}
	return nil
}

// validateAddServiceFee validates the AddServiceFee param
func ValidateAddServiceFee(v interface{}) error {
	addServiceFeeCoin, ok := v.(*cosmostypes.Coin)
	if !ok {
		return ErrServiceInvalidServiceFee.Wrapf("invalid parameter type: %T", v)
	}

	if addServiceFeeCoin == nil {
		return ErrServiceInvalidServiceFee.Wrap("missing proof_submission_fee")
	}

	if addServiceFeeCoin.Denom != volatile.DenomuPOKT {
		return ErrServiceInvalidServiceFee.Wrapf("invalid coin denom: %s", addServiceFeeCoin.Denom)
	}

	// TODO_MAINNET: Look into better validation
	if addServiceFeeCoin.Amount.LT(DefaultAddServiceFee.Amount) {
		return ErrServiceInvalidServiceFee.Wrapf(
			"AddServiceFee param is below minimum value %s: got %s",
			DefaultAddServiceFee,
			addServiceFeeCoin,
		)
	}

	return nil
}
