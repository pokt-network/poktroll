package types

// This file is in place to declare the package for dynamically generated protobufs

// ClaimProofStage is a string enum which represents a stage of a claim proof lifecycle.
type ClaimProofStage = string

const (
	ClaimProofStageClaimed = ClaimProofStage("claimed")
	ClaimProofStageProven  = ClaimProofStage("proven")
	ClaimProofStageSettled = ClaimProofStage("settled")
	ClaimProofStageExpired = ClaimProofStage("expired")
)

// ProofRequirementReason is a string enum which represents whether
// a proof is required, and why, if it is.
type ProofRequirementReason = string

const (
	ProofNotRequired                    = ProofRequirementReason("not_required")
	ProofRequirementReasonProbabilistic = ProofRequirementReason("probabilistic_selection")
	ProofRequirementReasonThreshold     = ProofRequirementReason("above_compute_unit_threshold")
)
