package proof

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// DefaultTestServiceID is the default test service ID
	DefaultTestServiceID = "svc1"
)

// BaseClaim returns a base (default, example, etc..) claim with the given
// service ID, app address, supplier address and num relays that can be used for testing.
func BaseClaim(serviceId, appAddr, supplierAddr string, numRelays uint64) prooftypes.Claim {
	computeUnitsPerRelay := uint64(1)
	sum := numRelays * computeUnitsPerRelay
	return prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: appAddr,
			Service: &sharedtypes.Service{
				Id:                   serviceId,
				ComputeUnitsPerRelay: computeUnitsPerRelay,
				OwnerAddress:         sample.AccAddress(), // This may need to be an input param in the future.
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		//
		RootHash: SmstRootWithSumAndCount(sum, numRelays),
	}
}

// ClaimWithRandomHash returns a claim with a random SMST root hash with the given
// app address, supplier address, and num relays that can be used for testing.
// Each claim generated this way will have a random chance to require a proof via
// probabilistic selection.
func ClaimWithRandomHash(t *testing.T, appAddr, supplierAddr string, numRelays uint64) prooftypes.Claim {
	claim := BaseClaim(DefaultTestServiceID, appAddr, supplierAddr, numRelays)
	claim.RootHash = RandSmstRootWithSumAndCount(t, numRelays, numRelays)
	return claim
}

// SmstRootWithSumAndCount returns a SMST root with the given sum and relay count.
func SmstRootWithSumAndCount(sum, count uint64) smt.MerkleSumRoot {
	root := [protocol.TrieRootSize]byte{}
	return encodeSmstRoot(root, sum, count)
}

// RandSmstRootWithSumAndCount returns a randomized SMST root with the given sum
// and count that can be used for testing. Randomizing the root is a simple way to
// randomize test claim hashes for testing proof requirement cases.
func RandSmstRootWithSumAndCount(t *testing.T, sum, count uint64) smt.MerkleSumRoot {
	t.Helper()

	root := [protocol.TrieRootSize]byte{}

	// Only populate the first 32 bytes with random data, leaving the rest to the sum and relay count.
	_, err := rand.Read(root[:protocol.TrieHasherSize]) // TODO_IMPROVE: We need a deterministic pseudo-random source.
	require.NoError(t, err)

	return encodeSmstRoot(root, sum, count)
}

// encodeSmstRoot returns a copy of the given root with the sum and count binary
// encoded and appended to the end.
// TODO_MAINNET: Revisit if the SMT should be big or little Endian. Refs:
// https://github.com/pokt-network/smt/pull/46#discussion_r1636975124
// https://github.com/pokt-network/smt/blob/ea585c6c3bc31c804b6bafa83e985e473b275580/smst.go#L23C10-L23C76
func encodeSmstRoot(root [protocol.TrieRootSize]byte, sum, count uint64) smt.MerkleSumRoot {
	encodedRoot := make([]byte, protocol.TrieRootSize)
	copy(encodedRoot, root[:])

	// Insert the sum into the root hash
	binary.BigEndian.PutUint64(encodedRoot[protocol.TrieHasherSize:], sum)
	// Insert the count into the root hash
	binary.BigEndian.PutUint64(encodedRoot[protocol.TrieHasherSize+protocol.TrieRootSumSize:], count)

	return encodedRoot
}
