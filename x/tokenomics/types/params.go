package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMintAllocationPercentages     = []byte("MintAllocationPercentages")
	ParamMintAllocationPercentages   = "mint_allocation_percentages"
	DefaultMintAllocationPercentages = MintAllocationPercentages{
		Dao:         0.1,
		Proposer:    0.05,
		Supplier:    0.7,
		SourceOwner: 0.15,
		Application: 0.0,
	}
	KeyDaoRewardAddress   = []byte("DaoRewardAddress")
	ParamDaoRewardAddress = "dao_reward_address"
	// DefaultDaoRewardAddress is the localnet DAO account address as specified in the config.yml.
	// It is only used in tests.
	DefaultDaoRewardAddress        = "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
	KeyGlobalInflationPerClaim     = []byte("GlobalInflationPerClaim")
	ParamGlobalInflationPerClaim   = "global_inflation_per_claim"
	DefaultGlobalInflationPerClaim = float64(0.1)

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	mintAllocationPercentages MintAllocationPercentages,
	daoRewardAddress string,
	globalInflationPerClaim float64,
) Params {
	return Params{
		MintAllocationPercentages: mintAllocationPercentages,
		DaoRewardAddress:          daoRewardAddress,
		GlobalInflationPerClaim:   globalInflationPerClaim,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMintAllocationPercentages,
		DefaultDaoRewardAddress,
		DefaultGlobalInflationPerClaim,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMintAllocationPercentages,
			&p.MintAllocationPercentages,
			ValidateMintAllocationPercentages,
		),
		paramtypes.NewParamSetPair(
			KeyMintAllocationPercentages,
			&p.MintAllocationPercentages,
			ValidateMintAllocationPercentages,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	if err := ValidateMintAllocationPercentages(params.MintAllocationPercentages); err != nil {
		return err
	}

	if err := ValidateDaoRewardAddress(params.DaoRewardAddress); err != nil {
		return err
	}

	if err := ValidateGlobalInflationPerClaim(params.GlobalInflationPerClaim); err != nil {
		return err
	}

	return nil
}

// ValidateMintAllocationDao validates the MintAllocationDao param.
func ValidateMintAllocationDao(mintAllocationDao any) error {
	return validateParamValueGTEZero(mintAllocationDao, "DAO")
}

// ValidateMintAllocationProposer validates the MintAllocationProposer param.
func ValidateMintAllocationProposer(mintAllocationProposer any) error {
	return validateParamValueGTEZero(mintAllocationProposer, "proposer")
}

// ValidateMintAllocationSupplier validates the MintAllocationSupplier param.
func ValidateMintAllocationSupplier(mintAllocationSupplier any) error {
	return validateParamValueGTEZero(mintAllocationSupplier, "supplier")
}

// ValidateMintAllocationSourceOwner validates the MintAllocationSourceOwner param.
func ValidateMintAllocationSourceOwner(mintAllocationSourceOwner any) error {
	return validateParamValueGTEZero(mintAllocationSourceOwner, "source owner")
}

// ValidateMintAllocationApplication validates the MintAllocationApplication param.
func ValidateMintAllocationApplication(mintAllocationApplication any) error {
	return validateParamValueGTEZero(mintAllocationApplication, "application")
}

func validateParamValueGTEZero(value any, actorName string) error {
	valueFloat, ok := value.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", value)
	}
	if valueFloat < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation to %s must be greater than or equal to 0: got %f", actorName, valueFloat)
	}
	return nil
}

func ValidateMintAllocationPercentages(mintAllocationPercentagesAny any) error {
	mintAllocationPercentages, ok := mintAllocationPercentagesAny.(MintAllocationPercentages)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type for mint_allocation_percentages: %T", mintAllocationPercentagesAny)
	}

	if err := ValidateMintAllocationDao(mintAllocationPercentages.Dao); err != nil {
		return err
	}

	if err := ValidateMintAllocationProposer(mintAllocationPercentages.Proposer); err != nil {
		return err
	}

	if err := ValidateMintAllocationSupplier(mintAllocationPercentages.Supplier); err != nil {
		return err
	}

	if err := ValidateMintAllocationSourceOwner(mintAllocationPercentages.SourceOwner); err != nil {
		return err
	}

	if err := ValidateMintAllocationApplication(mintAllocationPercentages.Application); err != nil {
		return err
	}

	if err := ValidateMintAllocationSum(mintAllocationPercentages); err != nil {
		return err
	}

	return nil
}

// ValidateMintAllocationSum validates that the sum of all actor mint allocation percentages is exactly 1.
func ValidateMintAllocationSum(mintAllocationPercentage MintAllocationPercentages) error {
	const epsilon = 1e-10 // Small epsilon value for floating-point comparison
	sum := mintAllocationPercentage.Sum()
	// TODO_IN_THIS_PR: is this safe to do? Added due to `mint allocation percentages do not add to 1.0: got 1.000000: the provided param is invalid`.
	if math.Abs(sum-1) > epsilon {
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

// ValidateGlobalInflationPerClaim validates the GlobalInflationPerClaim param.
func ValidateGlobalInflationPerClaim(GlobalInflationPerClaimAny any) error {
	GlobalInflationPerClaim, ok := GlobalInflationPerClaimAny.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", GlobalInflationPerClaimAny)
	}

	if GlobalInflationPerClaim < 0 {
		return ErrTokenomicsParamInvalid.Wrapf("GlobalInflationPerClaim must be greater than or equal to 0: %f", GlobalInflationPerClaim)
	}

	return nil
}
