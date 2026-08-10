package types

import (
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/poktroll/pkg/encoding"
)

// bigRatOne is the anti-collusion invariant's comparison bound (see Params.ValidateBasic).
// Package-level so the round-trip check does not allocate a new big.Rat per validation.
var bigRatOne = big.NewRat(1, 1)

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

	// Settlement budget redistribution: overservicing_bonus_multiplier bounds how far
	// above its guaranteed floor a supplier's settlement may be raised from unused budget.
	// 0 or 1 reproduce the legacy head-split cap exactly (no-op); n>1 allows up to n*floor.
	// The zero value is treated as 1 (legacy), so an unset/clobbered param is always benign.
	KeyOverservicingBonusMultiplier     = []byte("OverservicingBonusMultiplier")
	ParamOverservicingBonusMultiplier   = "overservicing_bonus_multiplier"
	DefaultOverservicingBonusMultiplier = uint64(1)

	_ paramtypes.ParamSet = (*Params)(nil)
)

// MaxOverservicingBonusMultiplier is the largest accepted overservicing_bonus_multiplier.
//
// Values at or above num_suppliers_per_session are already equivalent to "cap removed"
// — the application's committed per-session budget B (= numSuppliers * floor) binds
// before m does. So anything past that point buys nothing, while an accidental extra
// digit or a mis-encoded uint64 is indistinguishable from a deliberate choice. Bounding
// the param makes a fat-finger a loud validation error instead of a silent no-op.
//
// 1000 is ~an order of magnitude above any plausible num_suppliers_per_session, so it
// constrains nothing governance would legitimately want.
const MaxOverservicingBonusMultiplier = uint64(1000)

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
	overservicingBonusMultiplier uint64,
) Params {
	return Params{
		DaoRewardAddress:                daoRewardAddress,
		MintAllocationPercentages:       mintAllocationPercentages,
		GlobalInflationPerClaim:         globalInflationPerClaim,
		MintEqualsBurnClaimDistribution: mintEqualsBurnClaimDistribution,
		MintRatio:                       mintRatio,
		OverservicingBonusMultiplier:    overservicingBonusMultiplier,
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
		DefaultOverservicingBonusMultiplier,
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
		paramtypes.NewParamSetPair(
			KeyOverservicingBonusMultiplier,
			&p.OverservicingBonusMultiplier,
			ValidateOverservicingBonusMultiplier,
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

	if err := ValidateOverservicingBonusMultiplier(params.OverservicingBonusMultiplier); err != nil {
		return err
	}

	// If MintEqualsBurnClaimDistribution is zero-valued (e.g., because Ignite CLI couldn't parse it),
	// set it to the default value
	if params.MintEqualsBurnClaimDistribution.Sum() == 0 {
		params.MintEqualsBurnClaimDistribution = DefaultMintEqualsBurnClaimDistribution
	}

	// Anti-collusion invariant (settlement budget redistribution):
	// A colluding application+supplier burns X from its own application stake and
	// receives back mint_ratio * supplier_share * X as the supplier. For self-dealing
	// to be a losing trade, this round-trip factor MUST stay below 1. Today the
	// per-session head-split cap accidentally bounds collusion throughput; once that
	// cap is demoted to a floor, this invariant becomes the primary anti-collusion
	// mechanism, so it is enforced as a hard validation error here.
	//
	// Computed over big.Rat, NOT float64. A lone IEEE-754 multiply happens to be
	// bit-identical across platforms, but this is a consensus gate (it runs in the
	// v0.1.35 upgrade handler and on every MsgUpdateParam(s)), and the repo rule is
	// that float64 params are converted with encoding.Float64ToRat before any
	// arithmetic. Float64ToRat also goes through the decimal string, so 0.975 is
	// exactly 39/40 rather than the nearest binary double — the comparison against 1
	// is then exact instead of resting on the rounding of a product.
	mintRatioRat, err := encoding.Float64ToRat(params.MintRatio)
	if err != nil {
		return ErrTokenomicsParamInvalid.Wrapf("invalid mint_ratio: %s", err)
	}
	supplierShareRat, err := encoding.Float64ToRat(params.MintEqualsBurnClaimDistribution.Supplier)
	if err != nil {
		return ErrTokenomicsParamInvalid.Wrapf("invalid mint_equals_burn_claim_distribution.supplier: %s", err)
	}

	roundTripFactor := new(big.Rat).Mul(mintRatioRat, supplierShareRat)
	if roundTripFactor.Cmp(bigRatOne) >= 0 {
		return ErrTokenomicsParamInvalid.Wrapf(
			"anti-collusion invariant violated: mint_ratio (%s) * mint_equals_burn_claim_distribution.supplier (%s) = %s must be < 1",
			mintRatioRat.RatString(), supplierShareRat.RatString(), roundTripFactor.RatString(),
		)
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

// ValidateOverservicingBonusMultiplier validates the OverservicingBonusMultiplier param.
//
//   - 0 or 1 reproduce the legacy head-split cap exactly (no redistribution above the floor);
//     the zero value is treated as 1 at settlement so an unset/clobbered param stays benign,
//   - n > 1 bounds a supplier's settlement to n * floor (redistribution from unused budget),
//   - n > MaxOverservicingBonusMultiplier is REJECTED: it is economically identical to a
//     value at num_suppliers_per_session (the application budget B binds first), so such a
//     value is far more likely a typo than an intent. See MaxOverservicingBonusMultiplier.
func ValidateOverservicingBonusMultiplier(overservicingBonusMultiplierAny any) error {
	overservicingBonusMultiplier, ok := overservicingBonusMultiplierAny.(uint64)
	if !ok {
		return ErrTokenomicsParamInvalid.Wrapf("invalid parameter type: %T", overservicingBonusMultiplierAny)
	}

	if overservicingBonusMultiplier > MaxOverservicingBonusMultiplier {
		return ErrTokenomicsParamInvalid.Wrapf(
			"overservicing_bonus_multiplier (%d) must be <= %d; values at or above "+
				"num_suppliers_per_session already remove the cap in practice (the application's "+
				"per-session budget binds first), so a larger value has no effect and is most "+
				"likely a mistake",
			overservicingBonusMultiplier, MaxOverservicingBonusMultiplier,
		)
	}

	return nil
}
