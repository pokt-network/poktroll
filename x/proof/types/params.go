package types

import (
	"encoding/hex"

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

	KeyMinRelayDifficultyBits                   = []byte("MinRelayDifficultyBits")
	ParamRelayDifficultyTargetHash              = "relay_difficulty_target_hash"
	DefaultRelayDifficultyTargetHashHex         = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // all relays are payable
	DefaultRelayDifficultyTargetHash, _         = hex.DecodeString(DefaultRelayDifficultyTargetHashHex) // TODO_MAINNET(#142, #401): Determine the default value.
	KeyProofRequestProbability                  = []byte("ProofRequestProbability")
	ParamProofRequestProbability                = "proof_request_probability"
	DefaultProofRequestProbability      float32 = 0.25 // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md
	KeyProofRequirementThreshold                = []byte("ProofRequirementThreshold")
	ParamProofRequirementThreshold              = "proof_requirement_threshold"
	DefaultProofRequirementThreshold    uint64  = 20 // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md
	KeyProofMissingPenalty                      = []byte("ProofMissingPenalty")
	ParamProofMissingPenalty                    = "proof_missing_penalty"
	DefaultProofMissingPenalty                  = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(320)) // See: https://github.com/pokt-network/pocket-core/blob/staging/docs/proposals/probabilistic_proofs.md
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	relayDifficultyTargetHash []byte,
	proofRequestProbability float32,
	proofRequirementThreshold uint64,
	proofMissingPenalty *cosmostypes.Coin,
) Params {
	return Params{
		RelayDifficultyTargetHash: relayDifficultyTargetHash,
		ProofRequestProbability:   proofRequestProbability,
		ProofRequirementThreshold: proofRequirementThreshold,
		ProofMissingPenalty:       proofMissingPenalty,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultRelayDifficultyTargetHash,
		DefaultProofRequestProbability,
		DefaultProofRequirementThreshold,
		&DefaultProofMissingPenalty,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMinRelayDifficultyBits,
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

	return nil
}

// ValidateRelayDifficultyTargetHash validates the MinRelayDifficultyBits param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateRelayDifficultyTargetHash(v interface{}) error {
	targetHash, ok := v.([]byte)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if len(targetHash) != protocol.RelayHasherSize {
		return ErrProofParamInvalid.Wrapf(
			"invalid RelayDifficultyTargetHash: (%x); length wanted: %d; got: %d",
			targetHash,
			32,
			len(targetHash),
		)
	}

	return nil
}

// ValidateProofRequestProbability validates the ProofRequestProbability param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofRequestProbability(v interface{}) error {
	proofRequestProbability, ok := v.(float32)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if proofRequestProbability < 0 || proofRequestProbability > 1 {
		return ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", proofRequestProbability)
	}

	return nil
}

// ValidateProofRequirementThreshold validates the ProofRequirementThreshold param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofRequirementThreshold(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	return nil
}

// ValidateProofMissingPenalty validates the ProofMissingPenalty param.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateProofMissingPenalty(v interface{}) error {
	coin, ok := v.(*cosmostypes.Coin)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if coin == nil {
		return ErrProofParamInvalid.Wrap("missing proof_missing_penalty")
	}

	if coin.Denom != volatile.DenomuPOKT {
		return ErrProofParamInvalid.Wrapf("invalid coin denom: %s", coin.Denom)
	}

	return nil
}
