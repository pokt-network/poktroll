package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// The proof requirement sample must be the same before and after proof validation.
func TestClaim_GetProofRequirementSampleValue_IgnoresProofValidationStatus(t *testing.T) {
	claim := &Claim{
		SupplierOperatorAddress: "pokt1supplier",
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      "pokt1app",
			ServiceId:               "svc",
			SessionId:               "session",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   20,
		},
		RootHash: make([]byte, 48),
	}
	seedBlockHash := []byte("seed block hash")

	expected, err := claim.GetProofRequirementSampleValue(seedBlockHash)
	require.NoError(t, err)

	for _, status := range []ClaimProofStatus{ClaimProofStatus_VALIDATED, ClaimProofStatus_INVALID} {
		claim.ProofValidationStatus = status
		sample, err := claim.GetProofRequirementSampleValue(seedBlockHash)
		require.NoError(t, err)
		require.Equal(t, expected, sample, status.String())
		// The claim itself is not mutated.
		require.Equal(t, status, claim.ProofValidationStatus)
	}
}
