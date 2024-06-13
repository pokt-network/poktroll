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
func BaseClaim(appAddr, supplierAddr string, sum uint64) prooftypes.Claim {
	return prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: appAddr,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svcName1",
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
	claim := BaseClaim(appAddr, supplierAddr, sum)
	claim.RootHash = RandSmstRootWithSum(t, sum)
	return claim
}

// SmstRootWithSum returns a SMST root with the given sum that can be used for
// testing.
func SmstRootWithSum(sum uint64) smt.MerkleRoot {
	root := make([]byte, 40)
	copy(root[:32], []byte("This is exactly 32 characters!!!"))
	binary.BigEndian.PutUint64(root[32:], sum)
	return smt.MerkleRoot(root)
}

// RandSmstRootWithSum returns a randomized SMST root with the given sum that
// can be used for testing. Randomizing the root is a simple way to randomize
// test claim hashes for testing proof requirement cases.
func RandSmstRootWithSum(t *testing.T, sum uint64) smt.MerkleRoot {
	t.Helper()

	root := make([]byte, 40)
	// Only populate the first 32 bytes with random data, leave the last 8 bytes for the sum.
	_, err := rand.Read(root[:32])
	require.NoError(t, err)

	binary.BigEndian.PutUint64(root[32:], sum)
	return smt.MerkleRoot(root)
}
