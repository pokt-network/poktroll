package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyEarlietClaimSubmissionOffset = []byte("EarlietClaimSubmissionOffset")
	// TODO: Determine the default value
	DefaultEarlietClaimSubmissionOffset int32 = 0
)

var (
	KeyEarliestProofSubmissionOffset = []byte("EarliestProofSubmissionOffset")
	// TODO: Determine the default value
	DefaultEarliestProofSubmissionOffset int32 = 0
)

var (
	KeyLatestClaimSubmissionBlocksInterval = []byte("LatestClaimSubmissionBlocksInterval")
	// TODO: Determine the default value
	DefaultLatestClaimSubmissionBlocksInterval int32 = 0
)

var (
	KeyLatestProofSubmissionBlocksInterval = []byte("LatestProofSubmissionBlocksInterval")
	// TODO: Determine the default value
	DefaultLatestProofSubmissionBlocksInterval int32 = 0
)

var (
	KeyClaimSubmissionBlocksWindow = []byte("ClaimSubmissionBlocksWindow")
	// TODO: Determine the default value
	DefaultClaimSubmissionBlocksWindow int32 = 0
)

var (
	KeyProofSubmissionBlocksWindow = []byte("ProofSubmissionBlocksWindow")
	// TODO: Determine the default value
	DefaultProofSubmissionBlocksWindow int32 = 0
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	earlietClaimSubmissionOffset int32,
	earliestProofSubmissionOffset int32,
	latestClaimSubmissionBlocksInterval int32,
	latestProofSubmissionBlocksInterval int32,
	claimSubmissionBlocksWindow int32,
	proofSubmissionBlocksWindow int32,
) Params {
	return Params{
		EarlietClaimSubmissionOffset:        earlietClaimSubmissionOffset,
		EarliestProofSubmissionOffset:       earliestProofSubmissionOffset,
		LatestClaimSubmissionBlocksInterval: latestClaimSubmissionBlocksInterval,
		LatestProofSubmissionBlocksInterval: latestProofSubmissionBlocksInterval,
		ClaimSubmissionBlocksWindow:         claimSubmissionBlocksWindow,
		ProofSubmissionBlocksWindow:         proofSubmissionBlocksWindow,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultEarlietClaimSubmissionOffset,
		DefaultEarliestProofSubmissionOffset,
		DefaultLatestClaimSubmissionBlocksInterval,
		DefaultLatestProofSubmissionBlocksInterval,
		DefaultClaimSubmissionBlocksWindow,
		DefaultProofSubmissionBlocksWindow,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyEarlietClaimSubmissionOffset, &p.EarlietClaimSubmissionOffset, validateEarlietClaimSubmissionOffset),
		paramtypes.NewParamSetPair(KeyEarliestProofSubmissionOffset, &p.EarliestProofSubmissionOffset, validateEarliestProofSubmissionOffset),
		paramtypes.NewParamSetPair(KeyLatestClaimSubmissionBlocksInterval, &p.LatestClaimSubmissionBlocksInterval, validateLatestClaimSubmissionBlocksInterval),
		paramtypes.NewParamSetPair(KeyLatestProofSubmissionBlocksInterval, &p.LatestProofSubmissionBlocksInterval, validateLatestProofSubmissionBlocksInterval),
		paramtypes.NewParamSetPair(KeyClaimSubmissionBlocksWindow, &p.ClaimSubmissionBlocksWindow, validateClaimSubmissionBlocksWindow),
		paramtypes.NewParamSetPair(KeyProofSubmissionBlocksWindow, &p.ProofSubmissionBlocksWindow, validateProofSubmissionBlocksWindow),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateEarlietClaimSubmissionOffset(p.EarlietClaimSubmissionOffset); err != nil {
		return err
	}

	if err := validateEarliestProofSubmissionOffset(p.EarliestProofSubmissionOffset); err != nil {
		return err
	}

	if err := validateLatestClaimSubmissionBlocksInterval(p.LatestClaimSubmissionBlocksInterval); err != nil {
		return err
	}

	if err := validateLatestProofSubmissionBlocksInterval(p.LatestProofSubmissionBlocksInterval); err != nil {
		return err
	}

	if err := validateClaimSubmissionBlocksWindow(p.ClaimSubmissionBlocksWindow); err != nil {
		return err
	}

	if err := validateProofSubmissionBlocksWindow(p.ProofSubmissionBlocksWindow); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateEarlietClaimSubmissionOffset validates the EarlietClaimSubmissionOffset param
func validateEarlietClaimSubmissionOffset(v interface{}) error {
	earlietClaimSubmissionOffset, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = earlietClaimSubmissionOffset

	return nil
}

// validateEarliestProofSubmissionOffset validates the EarliestProofSubmissionOffset param
func validateEarliestProofSubmissionOffset(v interface{}) error {
	earliestProofSubmissionOffset, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = earliestProofSubmissionOffset

	return nil
}

// validateLatestClaimSubmissionBlocksInterval validates the LatestClaimSubmissionBlocksInterval param
func validateLatestClaimSubmissionBlocksInterval(v interface{}) error {
	latestClaimSubmissionBlocksInterval, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = latestClaimSubmissionBlocksInterval

	return nil
}

// validateLatestProofSubmissionBlocksInterval validates the LatestProofSubmissionBlocksInterval param
func validateLatestProofSubmissionBlocksInterval(v interface{}) error {
	latestProofSubmissionBlocksInterval, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = latestProofSubmissionBlocksInterval

	return nil
}

// validateClaimSubmissionBlocksWindow validates the ClaimSubmissionBlocksWindow param
func validateClaimSubmissionBlocksWindow(v interface{}) error {
	claimSubmissionBlocksWindow, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = claimSubmissionBlocksWindow

	return nil
}

// validateProofSubmissionBlocksWindow validates the ProofSubmissionBlocksWindow param
func validateProofSubmissionBlocksWindow(v interface{}) error {
	proofSubmissionBlocksWindow, ok := v.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = proofSubmissionBlocksWindow

	return nil
}
