package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMintAllocationDao             = []byte("MintAllocationDao")
	ParamMintAllocationDao           = "mint_allocation_dao"
	DefaultMintAllocationDao float32 = 0.1

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	mintAllocationDao float32,
) Params {
	return Params{
		MintAllocationDao: mintAllocationDao,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMintAllocationDao)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMintAllocationDao,
			&p.MintAllocationDao,
			ValidateMintAllocationDao,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	if err := ValidateMintAllocationDao(params.MintAllocationDao); err != nil {
		return err
	}

	return nil
}

// ValidateMintAllocationDao validates the MintAllocationDao param.
func ValidateMintAllocationDao(mintAllocationDao any) error {
	mintAllocationDaoFloat, ok := mintAllocationDao.(float32)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationDao)
	}

	if mintAllocationDaoFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to DAO must be greater than or equal to 0: got %f", mintAllocationDaoFloat)
	}

	return nil
}
