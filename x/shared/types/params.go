package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNumBlocksPerSession             = 4
	ParamNumBlocksPerSession               = "num_blocks_per_session"
	DefaultGracePeriodEndOffsetBlocks      = 1
	ParamGracePeriodEndOffsetBlocks        = "grace_period_end_offset_blocks"
	DefaultClaimWindowOpenOffsetBlocks     = 2
	ParamClaimWindowOpenOffsetBlocks       = "claim_window_open_offset_blocks"
	DefaultClaimWindowCloseOffsetBlocks    = 4
	ParamClaimWindowCloseOffsetBlocks      = "claim_window_close_offset_blocks"
	DefaultProofWindowOpenOffsetBlocks     = 0
	ParamProofWindowOpenOffsetBlocks       = "proof_window_open_offset_blocks"
	DefaultProofWindowCloseOffsetBlocks    = 4
	ParamProofWindowCloseOffsetBlocks      = "proof_window_close_offset_blocks"
	DefaultSupplierUnbondingPeriodSessions = 4 // 4 sessions
	ParamSupplierUnbondingPeriodSessions   = "supplier_unbonding_period_sessions"
)

var (
	_                                  paramtypes.ParamSet = (*Params)(nil)
	KeyNumBlocksPerSession                                 = []byte("NumBlocksPerSession")
	KeyGracePeriodEndOffsetBlocks                          = []byte("GracePeriodEndOffsetBlocks")
	KeyClaimWindowOpenOffsetBlocks                         = []byte("ClaimWindowOpenOffsetBlocks")
	KeyClaimWindowCloseOffsetBlocks                        = []byte("ClaimWindowCloseOffsetBlocks")
	KeyProofWindowOpenOffsetBlocks                         = []byte("ProofWindowOpenOffsetBlocks")
	KeyProofWindowCloseOffsetBlocks                        = []byte("ProofWindowCloseOffsetBlocks")
	KeySupplierUnbondingPeriodSessions                     = []byte("SupplierUnbondingPeriodSessions")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		NumBlocksPerSession:             DefaultNumBlocksPerSession,
		ClaimWindowOpenOffsetBlocks:     DefaultClaimWindowOpenOffsetBlocks,
		ClaimWindowCloseOffsetBlocks:    DefaultClaimWindowCloseOffsetBlocks,
		ProofWindowOpenOffsetBlocks:     DefaultProofWindowOpenOffsetBlocks,
		ProofWindowCloseOffsetBlocks:    DefaultProofWindowCloseOffsetBlocks,
		GracePeriodEndOffsetBlocks:      DefaultGracePeriodEndOffsetBlocks,
		SupplierUnbondingPeriodSessions: DefaultSupplierUnbondingPeriodSessions,
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
			KeySupplierUnbondingPeriodSessions,
			&params.SupplierUnbondingPeriodSessions,
			ValidateSupplierUnbondingPeriodSessions,
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

	if err := ValidateSupplierUnbondingPeriodSessions(params.SupplierUnbondingPeriodSessions); err != nil {
		return err
	}

	if err := validateGracePeriodOffsetBlocksIsLessThanNumBlocksPerSession(params); err != nil {
		return err
	}

	if err := validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset(params); err != nil {
		return err
	}

	if err := validateSupplierUnbondingPeriodIsGreaterThanCumulativeProofWindowCloseBlocks(params); err != nil {
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

// ValidateSupplierUnbondingPeriodSession validates the SupplierUnbondingPeriodSessions
// governance parameter.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateSupplierUnbondingPeriodSessions(v interface{}) error {
	supplierUnbondingPeriodSessions, err := validateIsUint64(v)
	if err != nil {
		return err
	}

	if supplierUnbondingPeriodSessions < 1 {
		return ErrSharedParamInvalid.Wrapf("invalid SupplierUnbondingPeriodSessions: (%v)", supplierUnbondingPeriodSessions)
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

// validateSupplierUnbondingPeriodIsGreaterThanCumulativeProofWindowCloseBlocks
// validates that the SupplierUnbondingPeriodSession blocks is greater than the cumulative
// proof window close blocks.
// It ensures that a supplier cannot unbond before the pending claims are settled.
func validateSupplierUnbondingPeriodIsGreaterThanCumulativeProofWindowCloseBlocks(params *Params) error {
	cumulativeProofWindowCloseBlocks := GetSessionEndToProofWindowCloseBlocks(params)
	supplierUnbondingPeriodSessions := params.SupplierUnbondingPeriodSessions * params.NumBlocksPerSession

	if supplierUnbondingPeriodSessions < cumulativeProofWindowCloseBlocks {
		return ErrSharedParamInvalid.Wrapf(
			"SupplierUnbondingPeriodSessions (%v session) (%v blocks) must be greater than the cumulative ProofWindowCloseOffsetBlocks (%v)",
			params.SupplierUnbondingPeriodSessions,
			supplierUnbondingPeriodSessions,
			cumulativeProofWindowCloseBlocks,
		)
	}

	return nil
}

// GetSessionEndToProofWindowCloseBlocks returns the total number of blocks
// from the moment a session ends until the proof window closes.
// NB: Using shared.GetProofWindowCloseOffsetHeight is not possible because of the
// circular dependency between the shared and session modules.
func GetSessionEndToProofWindowCloseBlocks(params *Params) uint64 {
	return params.ClaimWindowOpenOffsetBlocks +
		params.ClaimWindowCloseOffsetBlocks +
		params.ProofWindowOpenOffsetBlocks +
		params.ProofWindowCloseOffsetBlocks
}
