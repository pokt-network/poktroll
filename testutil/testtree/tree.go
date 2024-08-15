package testtree

import (
	"context"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// NewFilledSessionTree creates a new session tree with numRelays of relays
// filled out using the request and response headers provided where every
// relay is signed by the supplier and application respectively.
func NewFilledSessionTree(
	ctx context.Context, t *testing.T,
	numRelays, computeUnitsPerRelay uint64,
	supplierKeyUid, supplierOperatorAddr string,
	sessionTreeHeader, reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) relayer.SessionTree {
	t.Helper()

	// Initialize an empty session tree with the given session header.
	sessionTree := NewEmptySessionTree(t, sessionTreeHeader, supplierOperatorAddr)

	// Add numRelays of relays to the session tree.
	FillSessionTree(
		ctx, t,
		sessionTree,
		numRelays, computeUnitsPerRelay,
		supplierKeyUid, supplierOperatorAddr,
		reqHeader, resHeader,
		keyRing,
		ringClient,
	)

	return sessionTree
}

// NewEmptySessionTree creates a new empty session tree with for given session.
func NewEmptySessionTree(
	t *testing.T,
	sessionTreeHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) relayer.SessionTree {
	t.Helper()

	// Create a temporary session tree store directory for persistence.
	testSessionTreeStoreDir, err := os.MkdirTemp("", "session_tree_store_dir")
	require.NoError(t, err)

	// Delete the temporary session tree store directory after the test completes.
	t.Cleanup(func() {
		_ = os.RemoveAll(testSessionTreeStoreDir)
	})

	accAddress := cosmostypes.MustAccAddressFromBech32(supplierOperatorAddr)

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := session.NewSessionTree(
		sessionTreeHeader,
		&accAddress,
		testSessionTreeStoreDir,
	)
	require.NoError(t, err)

	return sessionTree
}

// FillSessionTree fills the session tree with valid signed relays.
// A total of numRelays relays are added to the session tree with
// increasing weights (relay 1 has weight 1, relay 2 has weight 2, etc.).
func FillSessionTree(
	ctx context.Context, t *testing.T,
	sessionTree relayer.SessionTree,
	numRelays, computeUnitsPerRelay uint64,
	supplierOperatorKeyUid, supplierOperatorAddr string,
	reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) {
	t.Helper()

	for i := 0; i < int(numRelays); i++ {
		relay := testrelayer.NewSignedEmptyRelay(
			ctx, t,
			supplierOperatorKeyUid, supplierOperatorAddr,
			reqHeader, resHeader,
			keyRing,
			ringClient,
		)
		relayBz, err := relay.Marshal()
		require.NoError(t, err)

		relayKey, err := relay.GetHash()
		require.NoError(t, err)

		err = sessionTree.Update(relayKey[:], relayBz, computeUnitsPerRelay)
		require.NoError(t, err)
	}
}

// NewProof creates a new proof structure.
func NewProof(
	t *testing.T,
	supplierOperatorAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *prooftypes.Proof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &prooftypes.Proof{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		ClosestMerkleProof:      merkleProofBz,
	}
}

func NewClaim(
	t *testing.T,
	supplierOperatorAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	rootHash []byte,
) *prooftypes.Claim {
	// Create a new claim.
	return &prooftypes.Claim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		RootHash:                rootHash,
	}
}
