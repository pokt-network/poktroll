package telemetry

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
	"github.com/pokt-network/smt"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

const (
	eventTypeMetricKey = "event_type"
)

type ClaimProofStage = string

const (
	ClaimProofStageClaimed = ClaimProofStage("claimed")
	ClaimProofStageProven  = ClaimProofStage("proven")
	ClaimProofStageSettled = ClaimProofStage("settled")
	ClaimProofStageExpired = ClaimProofStage("expired")
)

type ProofRequirementReason = string

const (
	ProofNotRequired                    = ProofRequirementReason("not_required")
	ProofRequirementReasonProbabilistic = ProofRequirementReason("probabilistic_selection")
	ProofRequirementReasonThreshold     = ProofRequirementReason("above_compute_unit_threshold")
)

// EventSuccessCounter increments a counter with the given data type and success status.
func EventSuccessCounter(
	eventType string,
	getValue func() float32,
	isSuccessful func() bool,
) {
	successResult := strconv.FormatBool(isSuccessful())
	value := getValue()

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		value,
		[]metrics.Label{
			{Name: "type", Value: eventType},
			{Name: "is_successful", Value: successResult},
		},
	)
}

// ProofRequirementCounter increments a counter which tracks the number of claims
// which require proof for the given proof requirement reason (i.e. not required,
// probabilistic selection, above compute unit threshold).
func ProofRequirementCounter(
	reason ProofRequirementReason,
	getValue func() float32,
) {
	value := getValue()

	isRequired := strconv.FormatBool(reason != ProofNotRequired)

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		value,
		[]metrics.Label{
			{Name: "proof_required_reason", Value: reason},
			{Name: "is_required", Value: isRequired},
		},
	)
}

// ComputeUnitsCounter increments a counter which tracks the number of compute units
// which are represented by on-chain claims at the given lifecycle state (i.e. claimed,
// proven, settled).
func ComputeUnitsCounter(lifecycleStage ClaimProofStage, claim *prooftypes.Claim) {
	root := (smt.MerkleRoot)(claim.GetRootHash())
	computeUnitsFloat := float32(root.Sum())

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		computeUnitsFloat,
		[]metrics.Label{
			{Name: "unit", Value: "compute_units"},
			{Name: "claim_proof_lifecycle_stage", Value: lifecycleStage},
		},
	)
}

func ClaimCounter(
	lifecycleStage ClaimProofStage,
	getValue func() uint64,
) {
	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		float32(getValue()),
		[]metrics.Label{
			{Name: "unit", Value: "claims"},
			{Name: "claim_proof_lifecycle_stage", Value: lifecycleStage},
		},
	)
}
