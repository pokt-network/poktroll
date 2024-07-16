package proof

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BaseClaim returns a base (default, example, etc..) claim with the given app
// address, supplier address, and sum that can be used for testing.
// The service ID is optional and defaults to "svc1" if not provided.
func BaseClaim(appAddr, supplierAddr string, sum uint64, serviceId string) prooftypes.Claim {
	if serviceId == "" {
		serviceId = "svc11"
	}
	return prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: appAddr,
			Service: &sharedtypes.Service{
				Id: serviceId,
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: SmstRootWithSum(sum),
	}
}

// ClaimWithRandomHash returns a claim with a random SMST root hash with the given
// app address, supplier address, and sum that can be used for testing. Each claim
// generated this way will have a random chance to require a proof via probabilistic
// selection.
func ClaimWithRandomHash(t *testing.T, appAddr, supplierAddr string, sum uint64) prooftypes.Claim {
	claim := BaseClaim(appAddr, supplierAddr, sum, "")
	claim.RootHash = RandSmstRootWithSum(t, sum)
	return claim
}

// SmstRootWithSum returns a SMST root with the given sum and a default
// hard-coded count of 1.
// TODO_POTENTIAL_TECHDEBT: Note that the count is meant to represent the number
// of non-empty leaves in the tree, and may need become a parameter depending on
// how the tests evolve.
// TODO_MAINNET: Revisit if the SMT should be big or little Endian. Refs:
// https://github.com/pokt-network/smt/pull/46#discussion_r1636975124
// https://github.com/pokt-network/smt/blob/ea585c6c3bc31c804b6bafa83e985e473b275580/smst.go#L23C10-L23C76
func SmstRootWithSum(sum uint64) smt.MerkleRoot {
	root := [smt.SmstRootSizeBytes]byte{}
	// Insert the sum into the root hash
	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes:], sum)
	// Insert the count into the root hash
	// TODO_TECHDEBT: This is a hard-coded count of 1, but could be a parameter.
	// TODO_TECHDEBT: We are assuming the sum takes up 8 bytes.
	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes+8:], 1)
	return smt.MerkleRoot(root[:])
}

// RandSmstRootWithSum returns a randomized SMST root with the given sum that
// can be used for testing. Randomizing the root is a simple way to randomize
// test claim hashes for testing proof requirement cases.
func RandSmstRootWithSum(t *testing.T, sum uint64) smt.MerkleRoot {
	t.Helper()

	root := [smt.SmstRootSizeBytes]byte{}
	// Only populate the first 32 bytes with random data, leave the last 8 bytes for the sum.
	_, err := rand.Read(root[:smt.SmtRootSizeBytes]) //nolint:staticcheck // We need a deterministic pseudo-random source.
	require.NoError(t, err)

	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes:], sum)
	// Insert the count into the root hash
	// TODO_TECHDEBT: This is a hard-coded count of 1, but could be a parameter.
	// TODO_TECHDEBT: We are assuming the sum takes up 8 bytes.
	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes+8:], 1)
	return smt.MerkleRoot(root[:])
}
