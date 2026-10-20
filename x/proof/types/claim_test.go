package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
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

// The sample hashes only the submitted fields, byte-identical to hashing the
// whole claim as created, so existing claims keep their sample.
func TestClaim_GetProofRequirementSampleValue_HashesSubmittedFields(t *testing.T) {
	claim := &Claim{
		SupplierOperatorAddress: "pokt1supplier",
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      "pokt1app",
			ServiceId:               "svc",
			SessionId:               "session",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   20,
		},
		RootHash:              []byte("root hash"),
		ProofValidationStatus: ClaimProofStatus_PENDING_VALIDATION,
	}
	seedBlockHash := []byte("seed block hash")

	claimHash, err := claim.GetHash()
	require.NoError(t, err)
	expected := poktrand.SeededFloat64(append(claimHash, seedBlockHash...))

	claim.ProofValidationStatus = ClaimProofStatus_INVALID
	sample, err := claim.GetProofRequirementSampleValue(seedBlockHash)
	require.NoError(t, err)
	require.Equal(t, expected, sample)
}
