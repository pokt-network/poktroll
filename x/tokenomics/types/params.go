package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	KeyDaoRewardAddress                      = []byte("DaoRewardAddress")
	ParamDaoRewardAddress                    = "dao_reward_address"
	// DefaultDaoRewardAddress is the localnet DAO account address as specified in the config.yml.
	// It is only used in tests.
	DefaultDaoRewardAddress = "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"

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
	daoRewardAddress string,
) Params {
	return Params{
		MintAllocationDao:         mintAllocationDao,
		MintAllocationProposer:    mintAllocationProposer,
		MintAllocationSupplier:    mintAllocationSupplier,
		MintAllocationSourceOwner: mintAllocationSourceOwner,
		MintAllocationApplication: mintAllocationApplication,
		DaoRewardAddress:          daoRewardAddress,
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
		DefaultDaoRewardAddress,
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
		paramtypes.NewParamSetPair(
			KeyDaoRewardAddress,
			&p.DaoRewardAddress,
			ValidateDaoRewardAddress,
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

	if err := ValidateDaoRewardAddress(params.DaoRewardAddress); err != nil {
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

// ValidateMintAllocationSum validates that the sum of all actor mint allocations is exactly 1.
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

// ValidateDaoRewardAddress validates the DaoRewardAddress param.
func ValidateDaoRewardAddress(daoRewardAddress any) error {
	daoRewardAddressStr, ok := daoRewardAddress.(string)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", daoRewardAddress)
	}

	if _, err := sdk.AccAddressFromBech32(daoRewardAddressStr); err != nil {
		return ErrTokenomicsParamInvalid.Wrapf("invalid dao reward address %q: %s", daoRewardAddressStr, err)
	}

	return nil
}
