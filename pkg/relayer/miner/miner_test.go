//go:generate go run gen/gen_fixtures.go gen/template.go

// (see: https://pkg.go.dev/cmd/go/internal/generate)
// (see: https://go.googlesource.com/proposal/+/refs/heads/master/design/go-generate.md)

package miner_test

import (
	"context"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// testSvcId is the ID of the service used in the relays. It is used to initialize the tokenomics module that
// is used by the relay miner to fetch the relay difficulty target hash for the service corresponding to the relay requests.
// The fixtures generated by pkg/relayer/miner/gen/gen_fixtures.go use a const with the same name and value.
const testSvcId = "svc1"

var testRelayMiningTargetHash, _ = hex.DecodeString("0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

// TestMiner_MinedRelays constructs an observable of mined relays, through which
// it pipes pre-mined relay fixtures. It asserts that the observable only emits
// mined relays with difficulty equal to or greater than testTargetHash.
//
// To regenerate all fixtures, use `make go_testgen_fixtures`; to regenerate only this
// test's fixtures run `go generate ./pkg/relayer/miner/miner_test.go`.
func TestMiner_MinedRelays(t *testing.T) {
	var (
		minedRelayCounter                     = 0
		ctx                                   = context.Background()
		actualMinedRelaysMu                   sync.Mutex
		actualMinedRelays                     []*relayer.MinedRelay
		mockRelaysObs, relaysFixturePublishCh = channel.NewObservable[*servicetypes.Relay]()
		expectedMinedRelays                   = unmarshalHexMinedRelays(t, marshaledMinableRelaysHex)
	)

	testqueryclients.SetServiceRelayDifficultyTargetHash(t, testSvcId, testRelayMiningTargetHash)
	serviceQueryClientMock := testqueryclients.NewTestServiceQueryClient(t)
	relayMeterMock := newMockRelayMeter(t)
	blockClient := mockclient.NewMockBlockClient(gomock.NewController(t))
	blockClient.EXPECT().
		GetChainVersion().
		DoAndReturn(func() *version.Version {
			chainVersion, err := version.NewVersion("v0.1.25")
			require.NoError(t, err)
			return chainVersion
		}).
		AnyTimes()

	deps := depinject.Supply(serviceQueryClientMock, relayMeterMock, blockClient)
	mnr, err := miner.NewMiner(deps)
	require.NoError(t, err)

	minedRelays := mnr.MinedRelays(ctx, mockRelaysObs)
	minedRelaysObserver := minedRelays.Subscribe(ctx)

	// Subscribe to the mined relays observable and append them to the
	// actualMinedRelays slice asynchronously.
	go func() {
		for minedRelay := range minedRelaysObserver.Ch() {
			actualMinedRelaysMu.Lock()
			actualMinedRelays = append(actualMinedRelays, minedRelay)
			minedRelayCounter++
			actualMinedRelaysMu.Unlock()
		}
	}()

	// Publish unminable relay fixtures to the mock relays observable.
	publishRelayFixtures(t, marshaledUnminableRelaysHex, relaysFixturePublishCh)
	time.Sleep(100 * time.Millisecond)

	// Assert that no unminable relay fixtures were published to minedRelays.
	actualMinedRelaysMu.Lock()
	require.Empty(t, actualMinedRelays)
	actualMinedRelaysMu.Unlock()

	// Publish minable relay fixtures to the relay fixtures observable.
	publishRelayFixtures(t, marshaledMinableRelaysHex, relaysFixturePublishCh)
	time.Sleep(100 * time.Millisecond)

	// Assert that all minable relay fixtures were published to minedRelays.
	actualMinedRelaysMu.Lock()
	// NB: We are comparing the lengths of the expected and actual relays instead of
	// the actual structures to simplify debugging. When there is an error, the output
	// is incomprehensible. The developer is expected to debug if this fails due to
	// a non-flaky reason.
	require.Equal(t, len(expectedMinedRelays), len(actualMinedRelays), "TODO_FLAKY: Try re-running with 'go test -v -count=1 -run TestMiner_MinedRelays ./pkg/relayer/miner/...'")
	actualMinedRelaysMu.Unlock()
}

func publishRelayFixtures(
	t *testing.T,
	marshalledRelaysHex []string,
	mockRelaysPublishCh chan<- *servicetypes.Relay,
) {
	t.Helper()

	for _, marshalledRelayHex := range marshalledRelaysHex {
		relay := unmarshalHexRelay(t, marshalledRelayHex)

		mockRelaysPublishCh <- relay
	}
}

func unmarshalHexRelay(
	t *testing.T,
	marshalledHexRelay string,
) *servicetypes.Relay {
	t.Helper()

	relayBz, err := hex.DecodeString(marshalledHexRelay)
	require.NoError(t, err)

	var relay servicetypes.Relay
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	return &relay
}

func unmarshalHexMinedRelays(
	t *testing.T,
	marshalledHexMinedRelays []string,
) (relays []*relayer.MinedRelay) {
	t.Helper()

	for _, marshalledRelayHex := range marshalledHexMinedRelays {
		relays = append(relays, unmarshalHexMinedRelay(t, marshalledRelayHex))
	}
	return relays
}

func unmarshalHexMinedRelay(
	t *testing.T,
	marshalledHexMinedRelay string,
) *relayer.MinedRelay {
	t.Helper()

	relayBz, err := hex.DecodeString(marshalledHexMinedRelay)
	require.NoError(t, err)

	var relay servicetypes.Relay
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}

// newMockRelayMeter returns a mock RelayMeter that is used by the relay miner to claim and unclaim relays.
func newMockRelayMeter(t *testing.T) relayer.RelayMeter {
	t.Helper()

	ctrl := gomock.NewController(t)
	relayMeter := mockrelayer.NewMockRelayMeter(ctrl)

	relayMeter.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()
	relayMeter.EXPECT().IsOverServicing(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	relayMeter.EXPECT().SetNonApplicableRelayReward(gomock.Any(), gomock.Any()).AnyTimes()

	return relayMeter
}
