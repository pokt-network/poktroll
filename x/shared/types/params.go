package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNumBlocksPerSession          = 4
	ParamNumBlocksPerSession            = "num_blocks_per_session"
	DefaultGracePeriodEndOffsetBlocks   = 1
	ParamGracePeriodEndOffsetBlocks     = "grace_period_end_offset_blocks"
	DefaultClaimWindowOpenOffsetBlocks  = 1
	ParamClaimWindowOpenOffsetBlocks    = "claim_window_open_offset_blocks"
	DefaultClaimWindowCloseOffsetBlocks = 4
	ParamClaimWindowCloseOffsetBlocks   = "claim_window_close_offset_blocks"
	DefaultProofWindowOpenOffsetBlocks  = 0
	ParamProofWindowOpenOffsetBlocks    = "proof_window_open_offset_blocks"
	DefaultProofWindowCloseOffsetBlocks = 4
	ParamProofWindowCloseOffsetBlocks   = "proof_window_close_offset_blocks"
)

var (
	_                               paramtypes.ParamSet = (*Params)(nil)
	KeyNumBlocksPerSession                              = []byte("NumBlocksPerSession")
	KeyGracePeriodEndOffsetBlocks                       = []byte("GracePeriodEndOffsetBlocks")
	KeyClaimWindowOpenOffsetBlocks                      = []byte("ClaimWindowOpenOffsetBlocks")
	KeyClaimWindowCloseOffsetBlocks                     = []byte("ClaimWindowCloseOffsetBlocks")
	KeyProofWindowOpenOffsetBlocks                      = []byte("ProofWindowOpenOffsetBlocks")
	KeyProofWindowCloseOffsetBlocks                     = []byte("ProofWindowCloseOffsetBlocks")
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
	}
}

// Validate validates the set of params
func (params *Params) ValidateBasic() error {
	// TODO_BLOCKER(@bryanchriswhite): Add `SessionGracePeriodBlocks` as a shared param,
	// introduce validation, and ensure it is strictly less than num blocks per session.

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

	if err := validateClaimWindowOpenOffsetIsAtLeastGracePeriodEndOffset(params); err != nil {
		return err
	}

	return nil
}

// ValidateNumBlocksPerSession validates the NumBlocksPerSession param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateNumBlocksPerSession(v interface{}) error {
	numBlocksPerSession, ok := v.(uint64)
	if !ok {
		return ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if numBlocksPerSession < 1 {
		return ErrSharedParamInvalid.Wrapf("invalid NumBlocksPerSession: (%v)", numBlocksPerSession)
	}

	return nil
}

// ValidateClaimWindowOpenOffsetBlocks validates the ClaimWindowOpenOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateClaimWindowOpenOffsetBlocks(v interface{}) error {
	return validateIsUint64(v)
}

// ValidateClaimWindowCloseOffsetBlocks validates the ClaimWindowCloseOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateClaimWindowCloseOffsetBlocks(v interface{}) error {
	return validateIsUint64(v)
}

// ValidateProofWindowOpenOffsetBlocks validates the ProofWindowOpenOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofWindowOpenOffsetBlocks(v interface{}) error {
	return validateIsUint64(v)
}

// ValidateProofWindowCloseOffsetBlocks validates the ProofWindowCloseOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofWindowCloseOffsetBlocks(v interface{}) error {
	return validateIsUint64(v)
}

// ValidateGracePeriodEndOffsetBlocks validates the GracePeriodEndOffsetBlocks param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateGracePeriodEndOffsetBlocks(v interface{}) error {
	return validateIsUint64(v)
}

// validateIsUint64 returns an error if value is not type assertable to uint64.
func validateIsUint64(value any) error {
	_, ok := value.(uint64)
	if !ok {
		return ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", value)
	}

	return nil
}

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
