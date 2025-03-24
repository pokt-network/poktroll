package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/pocket/app/volatile"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	// TODO_MAINNET: Determine a sensible default value for the min stake amount.
	KeyMinStake     = []byte("MinStake")
	ParamMinStake   = "min_stake"
	DefaultMinStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(minStake *cosmostypes.Coin) Params {
	return Params{
		MinStake: minStake,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(&DefaultMinStake)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMinStake,
			&p.MinStake,
			ValidateMinStake,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := ValidateMinStake(p.MinStake); err != nil {
		return err
	}
	return nil
}

func ValidateMinStake(minStakeAny any) error {
	minStakeCoin, ok := minStakeAny.(*cosmostypes.Coin)
	if !ok {
		return ErrGatewayParamInvalid.Wrapf("invalid type for %s: %T; expected *cosmostypes.Coin", ParamMinStake, minStakeAny)
	}

	if minStakeCoin == nil {
		return ErrGatewayParamInvalid.Wrap("missing min_stake")
	}

	if minStakeCoin.Denom != volatile.DenomuPOKT {
		return ErrGatewayParamInvalid.Wrapf(
			"invalid min_stake denom %q; expected %q",
			minStakeCoin.Denom, volatile.DenomuPOKT,
		)
	}
	if minStakeCoin.IsZero() || minStakeCoin.IsNegative() {
		return ErrGatewayParamInvalid.Wrapf("invalid min stake amount must be greater than zero: %s", minStakeCoin)
	}

	return nil
}
