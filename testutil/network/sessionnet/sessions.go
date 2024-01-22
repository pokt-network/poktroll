package sessionnet

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetSession sends a query using the test CLI to get a session for the inputs provided.
func (memnet *inMemoryNetworkWithSessions) GetSession(
	t *testing.T,
	serviceId string,
	appAddr string,
) *sessiontypes.Session {
	t.Helper()
	ctx := context.TODO()
	net := memnet.GetNetwork(t)

	sessionQueryClient := sessiontypes.NewQueryClient(net.Validators[0].ClientCtx)
	res, err := sessionQueryClient.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		Service:            &sharedtypes.Service{Id: serviceId},
		// NB(#196): intentionally omitting BlockHeight (i.e. session start height)
		// as it is difficult to predict in the general case.
		// BlockHeight:
	})
	require.NoError(t, err)

	return res.GetSession()
}

// cliEncodeSessionHeader encodes the given session header as a base64-encoded
// string.
func cliEncodeSessionHeader(t *testing.T, sessionHeader *sessiontypes.SessionHeader) string {
	t.Helper()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(sessionHeader)
	return base64.StdEncoding.EncodeToString(sessionHeaderBz)
}

// newSessionTreeRoot creates and returns a new session tree with the given number
// of relays and session header. All SMT persistence is done in a temporary and
// is cleaned up when the test completes.
func newSessionTreeRoot(
	t *testing.T,
	numRelays int,
	sessionHeader *sessiontypes.SessionHeader,
) relayer.SessionTree {
	t.Helper()

	tmpSmtStorePath, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)

	// Ensure all persisted trees are removed after the test completes.
	t.Cleanup(func() {
		if err = os.RemoveAll(tmpSmtStorePath); err != nil {
			t.Logf("WARNING: failed to delete temporary SMT store path %s: %s", tmpSmtStorePath, err)
		}
	})

	// NB: This function is a required constructor argument but is only called at the
	// end of `sessionTree#Delete()`, which this test doesn't exercise.
	noop := func(header *sessiontypes.SessionHeader) {}
	sessionTree, err := session.NewSessionTree(sessionHeader, tmpSmtStorePath, noop)
	require.NoError(t, err)

	for i := 0; i < numRelays; i++ {
		// While these relays use the `MinedRelay` data structure, they are not
		// "mined" in the sense that their inclusion in the tree is guaranteed.
		// `MinedRelay` fixtures produced this way effectively have difficulty 0.
		relay := testrelayer.NewMinedRelay(
			t, sessionHeader.GetSessionStartBlockHeight(),
			sessionHeader.GetSessionEndBlockHeight(),
		)

		err := sessionTree.Update(relay.Hash, relay.Bytes, 1)
		require.NoError(t, err)

	}
	return sessionTree
}

// getSessionTreeRoot returns the root hash of the given sessionTree as a both a
// byte slice and a base64-encoded string.
func getEncodedSessionTreeRoot(
	t *testing.T,
	sessionTree relayer.SessionTree,
) ([]byte, string) {
	t.Helper()

	rootHashBz, err := sessionTree.Flush()
	require.NoError(t, err)

	rootHashEncoded := base64.StdEncoding.EncodeToString(rootHashBz)
	return rootHashBz, rootHashEncoded
}
