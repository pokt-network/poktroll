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
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_IN_THIS_COMMIT: refactor to be a method on inMemoryNetworkWithSessions.
//
// getSessionId sends a query using the test CLI to get a session for the inputs provided.
// It is assumed that the supplierAddr will be in that session based on the test design, but this
// is insured in this function before it's successfully returned.
func GetSessionId(
	t *testing.T,
	net *network.Network,
	serviceId string,
	appAddr string,
	supplierAddr string,
	sessionStartHeight int64,
) string {
	t.Helper()
	ctx := context.TODO()

	sessionQueryClient := types.NewQueryClient(net.Validators[0].ClientCtx)
	res, err := sessionQueryClient.GetSession(ctx, &types.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		Service:            &sharedtypes.Service{Id: serviceId},
		BlockHeight:        sessionStartHeight,
	})
	require.NoError(t, err)

	var found bool
	for _, supplier := range res.GetSession().GetSuppliers() {
		if supplier.GetAddress() == supplierAddr {
			found = true
			break
		}
	}
	require.Truef(t, found, "supplier address %s not found in session", supplierAddr)

	return res.Session.SessionId
}

// newSessionHeader returns a session header and a base64-encoded string of its
// JSON-serialized representation.
func NewSessionHeader(
	t *testing.T,
	memnet *inMemoryNetworkWithSessions,
	serviceId string,
	appAddr string,
	supplierAddr string,
	sessionStartHeight int64,
) (*types.SessionHeader, string) {
	t.Helper()

	sessionId := GetSessionId(
		t, memnet.GetNetwork(t),
		serviceId,
		appAddr,
		supplierAddr,
		sessionStartHeight,
	)

	sessionHeader := &types.SessionHeader{
		ApplicationAddress:      appAddr,
		SessionStartBlockHeight: sessionStartHeight,
		SessionId:               sessionId,
		SessionEndBlockHeight:   sessionStartHeight + int64(memnet.config.NumBlocksPerSession),
		Service:                 &sharedtypes.Service{Id: testServiceId},
	}
	return sessionHeader, cliEncodeSessionHeader(t, sessionHeader)
}

func cliEncodeSessionHeader(t *testing.T, sessionHeader *types.SessionHeader) string {
	t.Helper()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(sessionHeader)
	return base64.StdEncoding.EncodeToString(sessionHeaderBz)
}

// TODO_IN_THIS_COMMIT: godoc comment...
func newSessionTreeRoot(
	t *testing.T,
	numRelays int,
	sessionHeader *types.SessionHeader,
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

	// This function is a required constructor argument but is only called at the
	// end of `sessionTree#Delete()`, which this test doesn't exercise.
	noop := func(header *types.SessionHeader) {}
	sessionTree, err := session.NewSessionTree(sessionHeader, tmpSmtStorePath, noop)
	require.NoError(t, err)

	for i := 0; i < numRelays; i++ {
		// While these relays use the `MinedRelay` data structure, they are not
		// "mined" in the sense that their inclusion is dependent on their difficulty.
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

func getSessionTreeRoot(
	t *testing.T,
	sessionTree relayer.SessionTree,
) ([]byte, string) {
	t.Helper()

	rootHashBz, err := sessionTree.Flush()
	require.NoError(t, err)

	rootHashEncoded := base64.StdEncoding.EncodeToString(rootHashBz)
	return rootHashBz, rootHashEncoded
}
