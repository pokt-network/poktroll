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
	ClaimProofStageClaiming = ClaimProofStage("claimed")
	ClaimProofStageProving  = ClaimProofStage("proven")
	ClaimProofStageSettling = ClaimProofStage("settled")
)

type ProofRequirementReason = string

const (
	ProofNotRequired                    = ProofRequirementReason("not_required")
	ProofRequirementReasonProbabilistic = ProofRequirementReason("probabilistic")
	ProofRequirementReasonThreshold     = ProofRequirementReason("threshold")
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

func ComputeUnitsCounter(lifecycleStage ClaimProofStage, claim *prooftypes.Claim) {
	root := (smt.MerkleRoot)(claim.GetRootHash())
	computeUnitsFloat := float32(root.Sum())

	telemetry.IncrCounterWithLabels(
		[]string{eventTypeMetricKey},
		computeUnitsFloat,
		[]metrics.Label{
			{Name: "proof_claim_lifecycle_stage", Value: lifecycleStage},
		},
	)
}
