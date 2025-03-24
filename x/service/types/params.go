package types

import (
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/pocket/app/volatile"
)

// DefaultAddServiceFee is the default value for the add service fee
// parameter in the genesis state of the service module.
// TODO_MAINNET: Revisit default param values for service fee
var (
	_ paramtypes.ParamSet = (*Params)(nil)

	// TODO_MAINNET: Determine a sensible default/min values.

	KeyAddServiceFee       = []byte("AddServiceFee")
	ParamAddServiceFee     = "add_service_fee"
	MinAddServiceFee       = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000000000))
	KeyTargetNumRelays     = []byte("TargetNumRelays")
	ParamTargetNumRelays   = "target_num_relays"
	DefaultTargetNumRelays = uint64(10e4)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	addServiceFee *cosmostypes.Coin,
	targetNumRelays uint64,
) Params {
	return Params{
		AddServiceFee:   addServiceFee,
		TargetNumRelays: targetNumRelays,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		&MinAddServiceFee,
		DefaultTargetNumRelays,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAddServiceFee, &p.AddServiceFee, ValidateAddServiceFee),
		paramtypes.NewParamSetPair(KeyTargetNumRelays, &p.AddServiceFee, ValidateTargetNumRelays),
	}
}

// ValidateBasic validates the set of params
func (p Params) ValidateBasic() error {
	if err := ValidateAddServiceFee(p.AddServiceFee); err != nil {
		return err
	}

	if err := ValidateTargetNumRelays(p.TargetNumRelays); err != nil {
		return err
	}

	return nil
}

// ValidateAddServiceFee validates the AddServiceFee param
func ValidateAddServiceFee(addServiceFeeAny any) error {
	addServiceFee, ok := addServiceFeeAny.(*cosmostypes.Coin)
	if !ok {
		return ErrServiceParamInvalid.Wrapf("invalid parameter type: %T", addServiceFeeAny)
	}

	if addServiceFee == nil {
		return ErrServiceParamInvalid.Wrap("missing add_service_fee")
	}

	if addServiceFee.Denom != volatile.DenomuPOKT {
		return ErrServiceParamInvalid.Wrapf("invalid add_service_fee denom: %s", addServiceFee.Denom)
	}

	// TODO_MAINNET: Look into better validation
	if addServiceFee.Amount.LT(MinAddServiceFee.Amount) {
		return ErrServiceParamInvalid.Wrapf(
			"add_service_fee param is below minimum value %s: got %s",
			MinAddServiceFee,
			addServiceFee,
		)
	}

	return nil
}

// ValidateTargetNumRelays validates the TargetNumRelays param
func ValidateTargetNumRelays(targetNumRelaysAny any) error {
	targetNumRelays, ok := targetNumRelaysAny.(uint64)
	if !ok {
		return ErrServiceParamInvalid.Wrapf("invalid parameter type: %T", targetNumRelaysAny)
	}

	if targetNumRelays < 1 {
		return ErrServiceParamInvalid.Wrapf("target_num_relays must be greater than 0: got %d", targetNumRelays)
	}

	return nil
}
