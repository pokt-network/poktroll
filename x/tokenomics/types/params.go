package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMintAllocationDao                     = []byte("MintAllocationDao")
	ParamMintAllocationDao                   = "mint_allocation_dao"
	DefaultMintAllocationDao         float64 = 0.1
	KeyMintAllocationProposer                = []byte("MintAllocationProposer")
	ParamMintAllocationProposer              = "mint_allocation_proposer"
	DefaultMintAllocationProposer    float64 = 0.05
	KeyMintAllocationSupplier                = []byte("MintAllocationSupplier")
	ParamMintAllocationSupplier              = "mint_allocation_supplier"
	DefaultMintAllocationSupplier    float64 = 0.7
	KeyMintAllocationSourceOwner             = []byte("MintAllocationSourceOwner")
	ParamMintAllocationSourceOwner           = "mint_allocation_source_owner"
	DefaultMintAllocationSourceOwner float64 = 0.15
	KeyMintAllocationApplication             = []byte("MintAllocationApplication")
	ParamMintAllocationApplication           = "mint_allocation_application"
	DefaultMintAllocationApplication float64 = 0.0

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	mintAllocationDao,
	mintAllocationProposer,
	mintAllocationSupplier,
	mintAllocationSourceOwner,
	mintAllocationApplication float64,
) Params {
	return Params{
		MintAllocationDao:         mintAllocationDao,
		MintAllocationProposer:    mintAllocationProposer,
		MintAllocationSupplier:    mintAllocationSupplier,
		MintAllocationSourceOwner: mintAllocationSourceOwner,
		MintAllocationApplication: mintAllocationApplication,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMintAllocationDao,
		DefaultMintAllocationProposer,
		DefaultMintAllocationSupplier,
		DefaultMintAllocationSourceOwner,
		DefaultMintAllocationApplication,
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
		paramtypes.NewParamSetPair(
			KeyMintAllocationSupplier,
			&p.MintAllocationSupplier,
			ValidateMintAllocationSupplier,
		),
		paramtypes.NewParamSetPair(
			KeyMintAllocationSourceOwner,
			&p.MintAllocationSourceOwner,
			ValidateMintAllocationSourceOwner,
		),
		paramtypes.NewParamSetPair(
			KeyMintAllocationApplication,
			&p.MintAllocationApplication,
			ValidateMintAllocationApplication,
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

	if err := ValidateMintAllocationSupplier(params.MintAllocationSupplier); err != nil {
		return err
	}

	if err := ValidateMintAllocationSourceOwner(params.MintAllocationSourceOwner); err != nil {
		return err
	}

	if err := ValidateMintAllocationApplication(params.MintAllocationApplication); err != nil {
		return err
	}

	if err := ValidateMintAllocationSum(params); err != nil {
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

// ValidateMintAllocationSupplier validates the MintAllocationSupplier param.
func ValidateMintAllocationSupplier(mintAllocationSupplier any) error {
	mintAllocationSupplierFloat, ok := mintAllocationSupplier.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationSupplier)
	}

	if mintAllocationSupplierFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to supplier must be greater than or equal to 0: got %f", mintAllocationSupplierFloat)
	}

	return nil
}

// ValidateMintAllocationSourceOwner validates the MintAllocationSourceOwner param.
func ValidateMintAllocationSourceOwner(mintAllocationSourceOwner any) error {
	mintAllocationSourceOwnerFloat, ok := mintAllocationSourceOwner.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationSourceOwner)
	}

	if mintAllocationSourceOwnerFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to source owner must be greater than or equal to 0: got %f", mintAllocationSourceOwnerFloat)
	}

	return nil
}

// ValidateMintAllocationApplication validates the MintAllocationApplication param.
func ValidateMintAllocationApplication(mintAllocationApplication any) error {
	mintAllocationApplicationFloat, ok := mintAllocationApplication.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintAllocationApplication)
	}

	if mintAllocationApplicationFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to application must be greater than or equal to 0: got %f", mintAllocationApplicationFloat)
	}

	return nil
}

// TODO_IN_THIS_COMMIT: godoc...
func ValidateMintAllocationSum(params *Params) error {
	sum := params.MintAllocationDao +
		params.MintAllocationProposer +
		params.MintAllocationSupplier +
		params.MintAllocationSourceOwner +
		params.MintAllocationApplication

	if sum != 1 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation percentages do not add to 1.0: got %f", sum)
	}

	return nil
}
