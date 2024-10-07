package types

import (
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
)

var (
	_ client.ProofParams  = (*Params)(nil)
	_ paramtypes.ParamSet = (*Params)(nil)

	// TODO_TECHDEBT(#690): Delete this parameter and just use "protocol.BaseRelayDifficultyHashBz"
	// as a hard-coded version. This param was originally intended to be a "minimum",
	// but the minimum equivalent of "0" in this case is "BaseRelayDifficultyHashBz".
	KeyRelayDifficultyTargetHash     = []byte("RelayDifficultyTargetHash")
	ParamRelayDifficultyTargetHash   = "relay_difficulty_target_hash"
	DefaultRelayDifficultyTargetHash = protocol.BaseRelayDifficultyHashBz

	// TODO_BETA(@red-0ne): Iterate on the parameters below by adding unit suffixes and
	// consider having the proof_requirement_threshold to be a function of the supplier's stake amount.

	KeyProofRequestProbability             = []byte("ProofRequestProbability")
	ParamProofRequestProbability           = "proof_request_probability"
	DefaultProofRequestProbability float32 = 0.25 // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md

	// The probabilistic proofs paper specifies a threshold of 20 POKT.
	// TODO_MAINNET(@Olshansk, @RawthiL): Figure out what this value should be.
	KeyProofRequirementThreshold     = []byte("ProofRequirementThreshold")
	ParamProofRequirementThreshold   = "proof_requirement_threshold"
	DefaultProofRequirementThreshold = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(20e6)) // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md

	// TODO_DISCUSS: Should ProofMissingPenalty be moved to the tokenomics module?
	KeyProofMissingPenalty   = []byte("ProofMissingPenalty")
	ParamProofMissingPenalty = "proof_missing_penalty"
	// As per the probabilistic proofs paper, the penalty for missing a proof is 320 POKT (i.e. 320e6 uPOKT).
	DefaultProofMissingPenalty = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(320e6)) // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md

	KeyProofSubmissionFee   = []byte("ProofSubmissionFee")
	ParamProofSubmissionFee = "proof_submission_fee"
	// TODO_MAINNET: Determine a sensible default value for the proof submission fee.
	// MinProofSubmissionFee is the default and minimum fee for submitting a proof.
	MinProofSubmissionFee = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000000))
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	relayDifficultyTargetHash []byte,
	proofRequestProbability float32,
	proofRequirementThreshold *cosmostypes.Coin,
	proofMissingPenalty *cosmostypes.Coin,
	proofSubmissionFee *cosmostypes.Coin,
) Params {
	return Params{
		RelayDifficultyTargetHash: relayDifficultyTargetHash,
		ProofRequestProbability:   proofRequestProbability,
		ProofRequirementThreshold: proofRequirementThreshold,
		ProofMissingPenalty:       proofMissingPenalty,
		ProofSubmissionFee:        proofSubmissionFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultRelayDifficultyTargetHash,
		DefaultProofRequestProbability,
		&DefaultProofRequirementThreshold,
		&DefaultProofMissingPenalty,
		&MinProofSubmissionFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyRelayDifficultyTargetHash,
			&p.RelayDifficultyTargetHash,
			ValidateRelayDifficultyTargetHash,
		),
		paramtypes.NewParamSetPair(
			KeyProofRequestProbability,
			&p.ProofRequestProbability,
			ValidateProofRequestProbability,
		),
		paramtypes.NewParamSetPair(
			KeyProofRequirementThreshold,
			&p.ProofRequirementThreshold,
			ValidateProofRequirementThreshold,
		),
		paramtypes.NewParamSetPair(
			KeyProofMissingPenalty,
			&p.ProofMissingPenalty,
			ValidateProofMissingPenalty,
		),
		paramtypes.NewParamSetPair(
			KeyProofSubmissionFee,
			&p.ProofSubmissionFee,
			ValidateProofSubmissionFee,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	// Validate the ComputeUnitsToTokensMultiplier
	if err := ValidateRelayDifficultyTargetHash(params.RelayDifficultyTargetHash); err != nil {
		return err
	}

	if err := ValidateProofRequestProbability(params.ProofRequestProbability); err != nil {
		return err
	}

	if err := ValidateProofRequirementThreshold(params.ProofRequirementThreshold); err != nil {
		return err
	}

	if err := ValidateProofMissingPenalty(params.ProofMissingPenalty); err != nil {
		return err
	}

	if err := ValidateProofSubmissionFee(params.ProofSubmissionFee); err != nil {
		return err
	}

	return nil
}

// ValidateRelayDifficultyTargetHash validates the MinRelayDifficultyBits param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateRelayDifficultyTargetHash(relayDifficultyTargetHashAny any) error {
	relayDifficultyTargetHash, ok := relayDifficultyTargetHashAny.([]byte)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", relayDifficultyTargetHashAny)
	}

	if len(relayDifficultyTargetHash) != protocol.RelayHasherSize {
		return ErrProofParamInvalid.Wrapf(
			"invalid RelayDifficultyTargetHash: (%x); length wanted: %d; got: %d",
			relayDifficultyTargetHash,
			32,
			len(relayDifficultyTargetHash),
		)
	}

	return nil
}

// ValidateProofRequestProbability validates the ProofRequestProbability param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofRequestProbability(proofRequestProbabilityAny any) error {
	proofRequestProbability, ok := proofRequestProbabilityAny.(float32)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", proofRequestProbabilityAny)
	}

	if proofRequestProbability < 0 || proofRequestProbability > 1 {
		return ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", proofRequestProbability)
	}

	return nil
}

// ValidateProofRequirementThreshold validates the ProofRequirementThreshold param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofRequirementThreshold(proofRequirementThresholdAny any) error {
	proofRequirementThresholdCoin, ok := proofRequirementThresholdAny.(*cosmostypes.Coin)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", proofRequirementThresholdAny)
	}

	if proofRequirementThresholdCoin == nil {
		return ErrProofParamInvalid.Wrap("missing proof_requirement_threshold")
	}

	if proofRequirementThresholdCoin.Denom != volatile.DenomuPOKT {
		return ErrProofParamInvalid.Wrapf("invalid proof_requirement_threshold denom: %s", proofRequirementThresholdCoin.Denom)
	}

	if proofRequirementThresholdCoin.IsZero() || proofRequirementThresholdCoin.IsNegative() {
		return ErrProofParamInvalid.Wrapf("invalid proof_requirement_threshold amount: %s <= 0", proofRequirementThresholdCoin)
	}

	return nil
}

// ValidateProofMissingPenalty validates the ProofMissingPenalty param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofMissingPenalty(proofMissingPenaltyAny any) error {
	proofMissingPenaltyCoin, ok := proofMissingPenaltyAny.(*cosmostypes.Coin)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", proofMissingPenaltyAny)
	}

	if proofMissingPenaltyCoin == nil {
		return ErrProofParamInvalid.Wrap("missing proof_missing_penalty")
	}

	if proofMissingPenaltyCoin.Denom != volatile.DenomuPOKT {
		return ErrProofParamInvalid.Wrapf("invalid proof_missing_penalty denom: %s", proofMissingPenaltyCoin.Denom)
	}

	if proofMissingPenaltyCoin.IsZero() || proofMissingPenaltyCoin.IsNegative() {
		return ErrProofParamInvalid.Wrapf("invalid proof_missing_penalty amount: %s <= 0", proofMissingPenaltyCoin)
	}

	return nil
}

// ValidateProofSubmissionFee validates the ProofSubmissionFee param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofSubmissionFee(proofSubmissionFeeAny any) error {
	submissionFeeCoin, ok := proofSubmissionFeeAny.(*cosmostypes.Coin)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", proofSubmissionFeeAny)
	}

	if submissionFeeCoin == nil {
		return ErrProofParamInvalid.Wrap("missing proof_submission_fee")
	}

	if submissionFeeCoin.Denom != volatile.DenomuPOKT {
		return ErrProofParamInvalid.Wrapf("invalid proof_submission_fee denom: %s", submissionFeeCoin.Denom)
	}

	if submissionFeeCoin.Amount.LT(MinProofSubmissionFee.Amount) {
		return ErrProofParamInvalid.Wrapf(
			"proof_submission_fee is below minimum value %s: got %s",
			MinProofSubmissionFee,
			submissionFeeCoin,
		)
	}

	return nil
}
