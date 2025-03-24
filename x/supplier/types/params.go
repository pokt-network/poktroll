package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/pocket/app/volatile"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyMinStake   = []byte("MinStake")
	ParamMinStake = "min_stake"
	// TODO_MAINNET: Determine the default value.
	DefaultMinStake = cosmostypes.NewInt64Coin("upokt", 1000000) // 1 POKT
	KeyStakingFee   = []byte("StakingFee")
	ParamStakingFee = "staking_fee"
	// TODO_MAINNET: Determine the default value.
	DefaultStakingFee = cosmostypes.NewInt64Coin("upokt", 1) // 1 uPOKT
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minStake *cosmostypes.Coin,
	stakingFee *cosmostypes.Coin,
) Params {
	return Params{
		MinStake:   minStake,
		StakingFee: stakingFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		&DefaultMinStake,
		&DefaultStakingFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMinStake,
			&p.MinStake,
			ValidateMinStake,
		),
		paramtypes.NewParamSetPair(
			KeyStakingFee,
			&p.StakingFee,
			ValidateStakingFee,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := ValidateMinStake(p.MinStake); err != nil {
		return err
	}

	if err := ValidateStakingFee(p.StakingFee); err != nil {
		return err
	}

	return nil
}

// ValidateMinStake validates the MinStake param.
func ValidateMinStake(minStakeAny any) error {
	minStakeCoin, err := paramAsPositiveuPOKT(minStakeAny, "min_stake")
	if err != nil {
		return err
	}

	if minStakeCoin.IsZero() {
		return ErrSupplierParamInvalid.Wrapf("min_stake amount must be greater than 0: got %s", minStakeCoin)
	}

	return nil
}

// ValidateStakingFee validates the StakingFee param.
func ValidateStakingFee(stakingFeeAny any) error {
	if _, err := paramAsPositiveuPOKT(stakingFeeAny, "staking_fee"); err != nil {
		return err
	}

	return nil
}

// paramAsPositiveuPOKT checks that paramAny is a *cosmostypes.Coin and that its
// amount is positive, returning an error if either is not the case.
func paramAsPositiveuPOKT(paramAny any, paramName string) (*cosmostypes.Coin, error) {
	paramCoin, ok := paramAny.(*cosmostypes.Coin)
	if !ok {
		return nil, ErrSupplierParamInvalid.Wrapf("invalid parameter type: %T", paramAny)
	}

	if paramCoin == nil {
		return nil, ErrSupplierParamInvalid.Wrapf("missing param")
	}

	if paramCoin.Denom != volatile.DenomuPOKT {
		return nil, ErrSupplierParamInvalid.Wrapf(
			"invalid %s denom %q; expected %q",
			paramName, paramCoin.Denom, volatile.DenomuPOKT,
		)
	}

	if paramCoin.IsNegative() {
		return nil, ErrSupplierParamInvalid.Wrapf("%s amount must be positive: got %s", paramName, paramCoin)
	}

	return paramCoin, nil
}
