package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMintAllocationDao                  = []byte("MintAllocationDao")
	ParamMintAllocationDao                = "mint_allocation_dao"
	DefaultMintAllocationDao      float64 = 0.1
	KeyMintAllocationProposer             = []byte("MintAllocationProposer")
	ParamMintAllocationProposer           = "mint_allocation_proposer"
	DefaultMintAllocationProposer float64 = 0.05

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	mintAllocationDao,
	mintAllocationProposer float64,
) Params {
	return Params{
		MintAllocationDao:      mintAllocationDao,
		MintAllocationProposer: mintAllocationProposer,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMintAllocationDao,
		DefaultMintAllocationProposer,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMintAllocationDao,
			&p.MintAllocationDao,
			ValidateMintAllocationDao,
		),
		paramtypes.NewParamSetPair(
			KeyMintAllocationProposer,
			&p.MintAllocationProposer,
			ValidateMintAllocationProposer,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	if err := ValidateMintAllocationDao(params.MintAllocationDao); err != nil {
		return err
	}

	if err := ValidateMintAllocationProposer(params.MintAllocationProposer); err != nil {
		return err
	}

	return nil
}

// ValidateMintAllocationDao validates the MintAllocationDao param.
func ValidateMintAllocationDao(mintAllocationDao any) error {
	mintAllocationDaoFloat, ok := mintAllocationDao.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationDao)
	}

	if mintAllocationDaoFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to DAO must be greater than or equal to 0: got %f", mintAllocationDaoFloat)
	}

	return nil
}

// ValidateMintAllocationProposer validates the MintAllocationProposer param.
func ValidateMintAllocationProposer(mintAllocationProposer any) error {
	mintAllocationProposerFloat, ok := mintAllocationProposer.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationProposer)
	}

	if mintAllocationProposerFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to proposer must be greater than or equal to 0: got %f", mintAllocationProposerFloat)
	}

	return nil
}
