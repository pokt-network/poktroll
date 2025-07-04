package testtree

import (
	"context"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
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
	sessionTree := NewEmptySessionTree(t, ctx, sessionTreeHeader, supplierOperatorAddr)

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
	ctx context.Context,
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

	logger := polylog.Ctx(ctx)

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := session.NewSessionTree(
		sessionTreeHeader,
		supplierOperatorAddr,
		testSessionTreeStoreDir,
		logger,
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
		relay := testrelayer.NewSignedRandRelay(
			ctx, t,
			supplierOperatorKeyUid, supplierOperatorAddr,
			reqHeader, resHeader,
			keyRing,
			ringClient,
		)

		// TODO(v0.1.26): Remove this if structure once all actors (miners, validators, etc.)
		// update to the version of the protocol that enforce the payload hash
		// and a nil payload.
		if len(relay.Res.PayloadHash) > 0 {
			relay.Res.Payload = nil
		}

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
	merkleCompactProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleCompactProof)

	// Serialize the closest merkle proof.
	merkleCompactProofBz, err := merkleCompactProof.Marshal()
	require.NoError(t, err)

	return &prooftypes.Proof{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		ClosestMerkleProof:      merkleCompactProofBz,
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
		ProofValidationStatus:   prooftypes.ClaimProofStatus_PENDING_VALIDATION,
	}
}
