package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// DAO TLM Params
	// DefaultDaoRewardAddress is the localnet DAO account address as specified in the config.yml.
	// It is only used in tests.
	KeyDaoRewardAddress     = []byte("DaoRewardAddress")
	ParamDaoRewardAddress   = "dao_reward_address"
	DefaultDaoRewardAddress = "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"

	// GlobalInflation TLM Params
	KeyGlobalInflationPerClaim     = []byte("GlobalInflationPerClaim")
	ParamGlobalInflationPerClaim   = "global_inflation_per_claim"
	DefaultGlobalInflationPerClaim = float64(0.1)

	// TODO_CONSIDERATION: Consider renaming this to GlobalInflationPerClaimDistribution
	// GlobalInflation Supporting TLM Params
	KeyMintAllocationPercentages     = []byte("MintAllocationPercentages")
	ParamMintAllocationPercentages   = "mint_allocation_percentages"
	DefaultMintAllocationPercentages = MintAllocationPercentages{
		Dao:         0.1,
		Proposer:    0.05,
		Supplier:    0.7,
		SourceOwner: 0.15,
		Application: 0.0,
	}

	// MintEqualsBurn Supporting TLM Params
	KeyMintEqualsBurnClaimDistribution     = []byte("MintEqualsBurnClaimDistribution")
	ParamMintEqualsBurnClaimDistribution   = "mint_equals_burn_claim_distribution"
	DefaultMintEqualsBurnClaimDistribution = MintEqualsBurnClaimDistribution{
		Dao:         0.1,
		Proposer:    0.05,
		Supplier:    0.7,
		SourceOwner: 0.15,
		Application: 0.0,
	}

	// PIP-41: MintRatio for deflationary mint mechanism
	// mint_ratio controls what proportion of burned tokens are minted (0.0 < mint_ratio <= 1.0)
	// A value of 0.975 means 97.5% of burned tokens are minted, 2.5% permanently removed
	KeyMintRatio     = []byte("MintRatio")
	ParamMintRatio   = "mint_ratio"
	DefaultMintRatio = float64(1.0) // Default: no deflation (mint equals burn)

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	daoRewardAddress string,
	mintAllocationPercentages MintAllocationPercentages,
	globalInflationPerClaim float64,
	mintEqualsBurnClaimDistribution MintEqualsBurnClaimDistribution,
	mintRatio float64,
) Params {
	return Params{
		DaoRewardAddress:                daoRewardAddress,
		MintAllocationPercentages:       mintAllocationPercentages,
		GlobalInflationPerClaim:         globalInflationPerClaim,
		MintEqualsBurnClaimDistribution: mintEqualsBurnClaimDistribution,
		MintRatio:                       mintRatio,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultDaoRewardAddress,
		DefaultMintAllocationPercentages,
		DefaultGlobalInflationPerClaim,
		DefaultMintEqualsBurnClaimDistribution,
		DefaultMintRatio,
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
			KeyDaoRewardAddress,
			&p.DaoRewardAddress,
			ValidateDaoRewardAddress,
		),
		paramtypes.NewParamSetPair(
			KeyGlobalInflationPerClaim,
			&p.GlobalInflationPerClaim,
			ValidateGlobalInflationPerClaim,
		),
		paramtypes.NewParamSetPair(
			KeyMintEqualsBurnClaimDistribution,
			&p.MintEqualsBurnClaimDistribution,
			ValidateMintEqualsBurnClaimDistribution,
		),
		paramtypes.NewParamSetPair(
			KeyMintRatio,
			&p.MintRatio,
			ValidateMintRatio,
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

	if err := ValidateMintEqualsBurnClaimDistribution(params.MintEqualsBurnClaimDistribution); err != nil {
		return err
	}

	if err := ValidateMintRatio(params.MintRatio); err != nil {
		return err
	}

	// If MintEqualsBurnClaimDistribution is zero-valued (e.g., because Ignite CLI couldn't parse it),
	// set it to the default value
	if params.MintEqualsBurnClaimDistribution.Sum() == 0 {
		params.MintEqualsBurnClaimDistribution = DefaultMintEqualsBurnClaimDistribution
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

// ValidateMintApplication validates the MintApplication param.
func ValidateMintApplication(mintApplication any) error {
	return validateParamValueGTEZero(mintApplication, "application")
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

	if err := ValidateMintApplication(mintAllocationPercentages.Application); err != nil {
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
	if math.Abs(sum-1) > epsilon {
		return ErrTokenomicsParamInvalid.Wrapf("mint allocation percentages do not add to 1.0: got %f instead. This is greater than the acceptable epsilon of %f", sum, epsilon)
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

// ValidateMintEqualsBurnClaimDistribution validates the MintEqualsBurnClaimDistribution param.
func ValidateMintEqualsBurnClaimDistribution(mintEqualsBurnClaimDistributionAny any) error {
	mintEqualsBurnClaimDistribution, ok := mintEqualsBurnClaimDistributionAny.(MintEqualsBurnClaimDistribution)
	if !ok {
		// If Ignite CLI can't parse the field correctly, this is still valid - the default will be used
		// This allows for graceful handling when config.yml contains the field but Ignite CLI can't parse complex nested structures
		return nil
	}

	// Validate individual percentages
	if err := validateParamValueGTEZero(mintEqualsBurnClaimDistribution.Dao, "DAO"); err != nil {
		return err
	}

	if err := validateParamValueGTEZero(mintEqualsBurnClaimDistribution.Proposer, "proposer"); err != nil {
		return err
	}

	if err := validateParamValueGTEZero(mintEqualsBurnClaimDistribution.Supplier, "supplier"); err != nil {
		return err
	}

	if err := validateParamValueGTEZero(mintEqualsBurnClaimDistribution.SourceOwner, "source owner"); err != nil {
		return err
	}

	if err := validateParamValueGTEZero(mintEqualsBurnClaimDistribution.Application, "application"); err != nil {
		return err
	}

	// Validate sum equals 1
	const epsilon = 1e-10 // Small epsilon value for floating-point comparison
	sum := mintEqualsBurnClaimDistribution.Sum()
	if math.Abs(sum-1) > epsilon {
		return ErrTokenomicsParamInvalid.Wrapf("mint equals burn claim distribution percentages do not add to 1.0: got %f", sum)
	}

	return nil
}

// ValidateMintRatio validates the MintRatio param.
// PIP-41: mint_ratio must be in range (0, 1] where:
// - 0 is exclusive (must mint something)
// - 1 is inclusive (can mint 100% = no deflation)
func ValidateMintRatio(mintRatioAny any) error {
	mintRatio, ok := mintRatioAny.(float64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", mintRatioAny)
	}

	if mintRatio <= 0 || mintRatio > 1 {
		return ErrTokenomicsParamInvalid.Wrapf("mint_ratio must be in range (0, 1]: got %f", mintRatio)
	}

	return nil
}
