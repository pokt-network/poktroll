package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeySupplierUnbondingPeriodBlocks            = []byte("SupplierUnbondingPeriodBlocks")
	ParamSupplierUnbondingPeriodBlocks          = "supplier_unbonding_period_blocks"
	DefaultSupplierUnbondingPeriodBlocks uint64 = 4 // TODO_MAINNET: Determine the default value.
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		SupplierUnbondingPeriodBlocks: DefaultSupplierUnbondingPeriodBlocks,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeySupplierUnbondingPeriodBlocks,
			&p.SupplierUnbondingPeriodBlocks,
			ValidateSupplierUnbondingPeriodBlocks,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	// Validate the SupplierUnbondingPeriodBlocks
	if err := ValidateSupplierUnbondingPeriodBlocks(p.SupplierUnbondingPeriodBlocks); err != nil {
		return err
	}

	return nil
}

// ValidateSupplierUnbondingPeriodBlocks validates the SupplierUnbondingPeriodBlocks
// governance parameter.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateSupplierUnbondingPeriodBlocks(v interface{}) error {
	supplierUnbondingPeriodBlocks, ok := v.(uint64)
	if !ok {
		return ErrSupplierParamsInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if supplierUnbondingPeriodBlocks <= 0 {
		return ErrSupplierParamsInvalid.Wrapf("invalid SupplierUnbondingPeriodBlocks: (%v)", supplierUnbondingPeriodBlocks)
	}

	return nil
}
