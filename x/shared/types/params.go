package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNumBlocksPerSession          = 4
	ParamNumBlocksPerSession            = "num_blocks_per_session"
	DefaultGracePeriodEndOffsetBlocks   = 1
	ParamGracePeriodEndOffsetBlocks     = "grace_period_end_offset_blocks"
	DefaultClaimWindowOpenOffsetBlocks  = 2
	ParamClaimWindowOpenOffsetBlocks    = "claim_window_open_offset_blocks"
	DefaultClaimWindowCloseOffsetBlocks = 4
	ParamClaimWindowCloseOffsetBlocks   = "claim_window_close_offset_blocks"
	DefaultProofWindowOpenOffsetBlocks  = 0
	ParamProofWindowOpenOffsetBlocks    = "proof_window_open_offset_blocks"
	DefaultProofWindowCloseOffsetBlocks = 4
	ParamProofWindowCloseOffsetBlocks   = "proof_window_close_offset_blocks"
	// The minimum value for the unbonding period in session number.
	// Its number of blocks must be greater than the cumulated proof window close blocks.
	DefaultSupplierUnbondingPeriod = 4
	ParamSupplierUnbondingPeriod   = "supplier_unbonding_period"
)

var (
	_                               paramtypes.ParamSet = (*Params)(nil)
	KeyNumBlocksPerSession                              = []byte("NumBlocksPerSession")
	KeyGracePeriodEndOffsetBlocks                       = []byte("GracePeriodEndOffsetBlocks")
	KeyClaimWindowOpenOffsetBlocks                      = []byte("ClaimWindowOpenOffsetBlocks")
	KeyClaimWindowCloseOffsetBlocks                     = []byte("ClaimWindowCloseOffsetBlocks")
	KeyProofWindowOpenOffsetBlocks                      = []byte("ProofWindowOpenOffsetBlocks")
	KeyProofWindowCloseOffsetBlocks                     = []byte("ProofWindowCloseOffsetBlocks")
	KeySupplierUnbondingPeriod                          = []byte("SupplierUnbondingPeriod")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		NumBlocksPerSession:          DefaultNumBlocksPerSession,
		ClaimWindowOpenOffsetBlocks:  DefaultClaimWindowOpenOffsetBlocks,
		ClaimWindowCloseOffsetBlocks: DefaultClaimWindowCloseOffsetBlocks,
		ProofWindowOpenOffsetBlocks:  DefaultProofWindowOpenOffsetBlocks,
		ProofWindowCloseOffsetBlocks: DefaultProofWindowCloseOffsetBlocks,
		GracePeriodEndOffsetBlocks:   DefaultGracePeriodEndOffsetBlocks,
		SupplierUnbondingPeriod:      DefaultSupplierUnbondingPeriod,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (params *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyNumBlocksPerSession,
			&params.NumBlocksPerSession,
			ValidateNumBlocksPerSession,
		),
		paramtypes.NewParamSetPair(
			KeyGracePeriodEndOffsetBlocks,
			&params.GracePeriodEndOffsetBlocks,
			ValidateGracePeriodEndOffsetBlocks,
		),
		paramtypes.NewParamSetPair(
			KeyClaimWindowOpenOffsetBlocks,
			&params.ClaimWindowOpenOffsetBlocks,
			ValidateClaimWindowOpenOffsetBlocks,
		),
		paramtypes.NewParamSetPair(
			KeyClaimWindowCloseOffsetBlocks,
			&params.ClaimWindowCloseOffsetBlocks,
			ValidateClaimWindowCloseOffsetBlocks,
		),
		paramtypes.NewParamSetPair(
			KeyProofWindowOpenOffsetBlocks,
			&params.ProofWindowOpenOffsetBlocks,
			ValidateProofWindowOpenOffsetBlocks,
		),
		paramtypes.NewParamSetPair(
			KeyProofWindowCloseOffsetBlocks,
			&params.ProofWindowCloseOffsetBlocks,
			ValidateProofWindowCloseOffsetBlocks,
		),
		paramtypes.NewParamSetPair(
			KeySupplierUnbondingPeriod,
			&params.SupplierUnbondingPeriod,
			ValidateProofWindowCloseOffsetBlocks,
		),
	}
}

// Validate validates the set of params
func (params *Params) ValidateBasic() error {
	if err := ValidateNumBlocksPerSession(params.NumBlocksPerSession); err != nil {
		return err
	}

	if err := ValidateClaimWindowOpenOffsetBlocks(params.ClaimWindowOpenOffsetBlocks); err != nil {
		return err
	}

	if err := ValidateClaimWindowCloseOffsetBlocks(params.ClaimWindowCloseOffsetBlocks); err != nil {
		return err
	}

	if err := ValidateProofWindowOpenOffsetBlocks(params.ProofWindowOpenOffsetBlocks); err != nil {
		return err
	}

	if err := ValidateProofWindowCloseOffsetBlocks(params.ProofWindowCloseOffsetBlocks); err != nil {
		return err
	}

	if err := ValidateGracePeriodEndOffsetBlocks(params.GracePeriodEndOffsetBlocks); err != nil {
		return err
	}

	if err := ValidateSupplierUnbondingPeriod(params.SupplierUnbondingPeriod); err != nil {
		return err
	}

	if err := validateGracePeriodOffsetBlocksIsLessThanNumBlocksPerSession(params); err != nil {
		return err
	}

	if err := validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset(params); err != nil {
		return err
	}

	if err := validateSupplierUnbondingPeriodIsGreaterThanCumulatedProofWindowCloseBlocks(params); err != nil {
		return err
	}

	return nil
}

// ValidateNumBlocksPerSession validates the NumBlocksPerSession param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateNumBlocksPerSession(v interface{}) error {
	numBlocksPerSession, err := validateIsUint64(v)
	if err != nil {
		return err
	}

	if numBlocksPerSession < 1 {
		return ErrSharedParamInvalid.Wrapf("invalid NumBlocksPerSession: (%v)", numBlocksPerSession)
	}

	return nil
}

// ValidateClaimWindowOpenOffsetBlocks validates the ClaimWindowOpenOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateClaimWindowOpenOffsetBlocks(v interface{}) error {
	_, err := validateIsUint64(v)
	return err
}

// ValidateClaimWindowCloseOffsetBlocks validates the ClaimWindowCloseOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateClaimWindowCloseOffsetBlocks(v interface{}) error {
	_, err := validateIsUint64(v)
	return err
}

// ValidateProofWindowOpenOffsetBlocks validates the ProofWindowOpenOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofWindowOpenOffsetBlocks(v interface{}) error {
	_, err := validateIsUint64(v)
	return err
}

// ValidateProofWindowCloseOffsetBlocks validates the ProofWindowCloseOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofWindowCloseOffsetBlocks(v interface{}) error {
	_, err := validateIsUint64(v)
	return err
}

// ValidateGracePeriodEndOffsetBlocks validates the GracePeriodEndOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateGracePeriodEndOffsetBlocks(v interface{}) error {
	_, err := validateIsUint64(v)
	return err
}

// ValidateSupplierUnbondingPeriodBlocks validates the SupplierUnbondingPeriod
// governance parameter.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateSupplierUnbondingPeriod(v interface{}) error {
	supplierUnbondingPeriod, err := validateIsUint64(v)
	if err != nil {
		return err
	}

	if supplierUnbondingPeriod < 1 {
		return ErrSharedParamInvalid.Wrapf("invalid SupplierUnbondingPeriodBlocks: (%v)", supplierUnbondingPeriod)
	}

	return nil
}

// validateIsUint64 returns the casted uin64 value or an error if value is not
// type assertable to uint64.
func validateIsUint64(value any) (uint64, error) {
	uint64Value, ok := value.(uint64)
	if !ok {
		return 0, ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", value)
	}

	return uint64Value, nil
}

// validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset validates that the ClaimWindowOpenOffsetBlocks
// is at least as big as GracePeriodEndOffsetBlocks. The claim window cannot open until the grace period ends
// to ensure that the seed for the earliest supplier claim commit height is only observed after the last relay
// for a given session could be serviced.
func validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset(params *Params) error {
	if params.ClaimWindowOpenOffsetBlocks < params.GracePeriodEndOffsetBlocks {
		return ErrSharedParamInvalid.Wrapf(
			"ClaimWindowOpenOffsetBlocks (%v) must be at least GracePeriodEndOffsetBlocks (%v)",
			params.ClaimWindowOpenOffsetBlocks,
			params.GracePeriodEndOffsetBlocks,
		)
	}
	return nil
}

// validateGracePeriodOffsetBlocksIsLessThanNumBlocksPerSession validates that the
// GracePeriodEndOffsetBlocks is less than NumBlocksPerSession; i.e., one session.
func validateGracePeriodOffsetBlocksIsLessThanNumBlocksPerSession(params *Params) error {
	if params.GracePeriodEndOffsetBlocks >= params.NumBlocksPerSession {
		return ErrSharedParamInvalid.Wrapf(
			"GracePeriodEndOffsetBlocks (%v) must be less than NumBlocksPerSession (%v)",
			params.GracePeriodEndOffsetBlocks,
			params.NumBlocksPerSession,
		)
	}
	return nil
}

// validateSupplierUnbondingPeriodIsGreaterThanCumulatedProofWindowCloseBlocks
// validates that the SupplierUnbondingPeriod blocks is greater than the cumulated
// proof window close blocks.
// It ensures that a supplier cannot unbond before the pending claims are settled.
func validateSupplierUnbondingPeriodIsGreaterThanCumulatedProofWindowCloseBlocks(params *Params) error {
	cumulatedProofWindowCloseBlocks := GetCumulatedProofWindowCloseBlocks(params)
	supplierUnbondingPeriodBlocks := params.SupplierUnbondingPeriod * params.NumBlocksPerSession

	if supplierUnbondingPeriodBlocks < cumulatedProofWindowCloseBlocks {
		return ErrSharedParamInvalid.Wrapf(
			"SupplierUnbondingPeriod (%v session) (%v blocks) must be greater than the cumulated ProofWindowCloseOffsetBlocks (%v)",
			params.SupplierUnbondingPeriod,
			supplierUnbondingPeriodBlocks,
			cumulatedProofWindowCloseBlocks,
		)
	}

	return nil
}

// GetCumulatedProofWindowCloseBlocks returns the total number of blocks required
// to pass the proof window close of a session.
// Using shared.GetProofWindowCloseOffsetHeight is not possible because of the
// circular dependency between the shared and session modules.
func GetCumulatedProofWindowCloseBlocks(params *Params) uint64 {
	return params.ClaimWindowOpenOffsetBlocks +
		params.ClaimWindowCloseOffsetBlocks +
		params.ProofWindowOpenOffsetBlocks +
		params.ProofWindowCloseOffsetBlocks
}
