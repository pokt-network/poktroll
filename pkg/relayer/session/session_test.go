package session_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testsupplier"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestRelayerSessionsManager_Start(t *testing.T) {
	const (
		sessionStartHeight = 1
		sessionEndHeight   = 2
	)
	var (
		zeroByteSlice = []byte{0}
		ctx           = context.Background()
	)

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 1)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, blocksObs)
	supplierClient := testsupplier.NewOneTimeClaimProofSupplierClient(ctx, t)

	deps := depinject.Supply(blockClient, supplierClient)
	storesDirectoryOpt := testrelayer.WithTempStoresDirectory(t)

	// Create a new relayer sessions manager.
	relayerSessionsManager, err := session.NewRelayerSessions(ctx, deps, storesDirectoryOpt)
	require.NoError(t, err)
	require.NotNil(t, relayerSessionsManager)

	// Pass a mined relays observable via InsertRelays.
	mrObs, minedRelaysPublishCh := channel.NewObservable[*relayer.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)
	relayerSessionsManager.InsertRelays(minedRelaysObs)

	// Start the relayer sessions manager.
	relayerSessionsManager.Start(ctx)

	// Publish a mined relay to the minedRelaysPublishCh to insert into the session tree.
	minedRelay := newMinedRelay(t, sessionStartHeight, sessionEndHeight)
	minedRelaysPublishCh <- minedRelay

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	// It should have created a session tree for the relay.
	time.Sleep(10 * time.Millisecond)

	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	noopBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionStartHeight)
	blockPublishCh <- noopBlock

	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	// TODO_TECHDEBT: assumes claiming at sessionEndHeight is valid. This will
	// likely change in future work.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionEndHeight)
	blockPublishCh <- triggerClaimBlock

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	// TODO_TECHDEBT: assumes proving at sessionEndHeight is valid. This will
	// likely change in future work.
	triggerProofBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionEndHeight+1)
	blockPublishCh <- triggerProofBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(250 * time.Millisecond)
}

// newMinedRelay returns a new mined relay with the given session start and end
// heights on the session header, and the bytes and hash fields populated.
func newMinedRelay(
	t *testing.T,
	sessionStartHeight int64,
	sessionEndHeight int64,
) *relayer.MinedRelay {
	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: &servicetypes.RelayRequestMetadata{
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: sessionStartHeight,
					SessionEndBlockHeight:   sessionEndHeight,
				},
			},
		},
		Res: &servicetypes.RelayResponse{},
	}

	// TODO_BLOCKER: use canonical codec to serialize the relay
	relayBz, err := relay.Marshal()
	require.NoError(t, err)

	relayHash := testrelayer.HashBytes(t, miner.DefaultRelayHasher, relayBz)

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}
