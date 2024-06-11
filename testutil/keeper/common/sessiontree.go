package common

import (
	"context"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newFilledSessionTree creates a new session tree with numRelays of relays
// filled out using the request and response headers provided where every
// relay is signed by the supplier and application respectively.
func NewFilledSessionTree(
	ctx context.Context, t *testing.T,
	numRelays uint,
	supplierKeyUid, supplierAddr string,
	sessionTreeHeader, reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) relayer.SessionTree {
	t.Helper()

	// Initialize an empty session tree with the given session header.
	sessionTree := NewEmptySessionTree(t, sessionTreeHeader)

	// Add numRelays of relays to the session tree.
	FillSessionTree(
		ctx, t,
		sessionTree, numRelays,
		supplierKeyUid, supplierAddr,
		reqHeader, resHeader,
		keyRing,
		ringClient,
	)

	return sessionTree
}

// newEmptySessionTree creates a new empty session tree with for given session.
func NewEmptySessionTree(
	t *testing.T,
	sessionTreeHeader *sessiontypes.SessionHeader,
) relayer.SessionTree {
	t.Helper()

	// Create a temporary session tree store directory for persistence.
	testSessionTreeStoreDir, err := os.MkdirTemp("", "session_tree_store_dir")
	require.NoError(t, err)

	// Delete the temporary session tree store directory after the test completes.
	t.Cleanup(func() {
		_ = os.RemoveAll(testSessionTreeStoreDir)
	})

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := session.NewSessionTree(
		sessionTreeHeader,
		testSessionTreeStoreDir,
		func(*sessiontypes.SessionHeader) {},
	)
	require.NoError(t, err)

	return sessionTree
}

// fillSessionTree fills the session tree with valid signed relays.
// A total of numRelays relays are added to the session tree with
// increasing weights (relay 1 has weight 1, relay 2 has weight 2, etc.).
func FillSessionTree(
	ctx context.Context, t *testing.T,
	sessionTree relayer.SessionTree,
	numRelays uint,
	supplierKeyUid, supplierAddr string,
	reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) {
	t.Helper()

	for i := 0; i < int(numRelays); i++ {
		relay := NewSignedEmptyRelay(
			ctx, t,
			supplierKeyUid, supplierAddr,
			reqHeader, resHeader,
			keyRing,
			ringClient,
		)
		relayBz, err := relay.Marshal()
		require.NoError(t, err)

		relayKey, err := relay.GetHash()
		require.NoError(t, err)

		relayWeight := uint64(i)

		err = sessionTree.Update(relayKey[:], relayBz, relayWeight)
		require.NoError(t, err)
	}
}
