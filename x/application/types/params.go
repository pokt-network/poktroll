package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/poktroll/app/volatile"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyMaxDelegatedGateways   = []byte("MaxDelegatedGateways")
	ParamMaxDelegatedGateways = "max_delegated_gateways"
	// TODO_MAINNET: Determine the default value
	DefaultMaxDelegatedGateways uint64 = 7
	KeyMinStake                        = []byte("MinStake")
	ParamMinStake                      = "min_stake"
	// TODO_MAINNET: Determine the default value
	DefaultMinStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000) // 1 POKT
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(maxDelegatedGateways uint64, minStake *cosmostypes.Coin) Params {
	return Params{
		MaxDelegatedGateways: maxDelegatedGateways,
		MinStake:             minStake,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMaxDelegatedGateways, &DefaultMinStake)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxDelegatedGateways, &p.MaxDelegatedGateways, ValidateMaxDelegatedGateways),
		paramtypes.NewParamSetPair(KeyMinStake, &p.MinStake, ValidateMinStake),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := ValidateMaxDelegatedGateways(p.MaxDelegatedGateways); err != nil {
		return err
	}

	if err := ValidateMinStake(p.MinStake); err != nil {
		return err
	}

	return nil
}

// ValidateMaxDelegatedGateways validates the MaxDelegatedGateways param
func ValidateMaxDelegatedGateways(maxDelegatedGatewaysAny any) error {
	maxDelegatedGateways, ok := maxDelegatedGatewaysAny.(uint64)
	if !ok {
		return ErrAppParamInvalid.Wrapf("invalid parameter type: %T", maxDelegatedGatewaysAny)
	}

	// Hard-coding a value of 1 because we never expect this to change.
	// If an application chooses to delegate, at least one is required.
	if maxDelegatedGateways < 1 {
		return ErrAppParamInvalid.Wrapf("max_delegated_gateways must be greater than 0: got %d", maxDelegatedGateways)
	}

	return nil
}

// ValidateMinStake validates the MinStake param
func ValidateMinStake(minStakeAny any) error {
	minStakeCoin, ok := minStakeAny.(*cosmostypes.Coin)
	if !ok {
		return ErrAppParamInvalid.Wrapf("invalid parameter type: %T", minStakeAny)
	}

	if minStakeCoin == nil {
		return ErrAppParamInvalid.Wrapf("missing min_stake")
	}

	if minStakeCoin.Denom != volatile.DenomuPOKT {
		return ErrAppParamInvalid.Wrapf(
			"invalid min_stake denom %q; expected %q",
			minStakeCoin.Denom, volatile.DenomuPOKT,
		)
	}

	if minStakeCoin.IsZero() || minStakeCoin.IsNegative() {
		return ErrAppParamInvalid.Wrapf("invalid min_stake amount: %s <= 0", minStakeCoin)
	}

	return nil
}
