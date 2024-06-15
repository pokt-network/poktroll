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

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const testDifficulty = 16

// TestMiner_MinedRelays constructs an observable of mined relays, through which
// it pipes pre-mined relay fixtures. It asserts that the observable only emits
// mined relays with difficulty equal to or greater than testDifficulty.
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

	mnr, err := miner.NewMiner(miner.WithDifficulty(testDifficulty))
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
	time.Sleep(time.Millisecond)

	// Assert that no unminable relay fixtures were published to minedRelays.
	actualMinedRelaysMu.Lock()
	require.Empty(t, actualMinedRelays)
	actualMinedRelaysMu.Unlock()

	// Publish minable relay fixtures to the relay fixtures observable.
	publishRelayFixtures(t, marshaledMinableRelaysHex, relaysFixturePublishCh)
	time.Sleep(time.Millisecond)

	// Assert that all minable relay fixtures were published to minedRelays.
	actualMinedRelaysMu.Lock()
	require.EqualValues(t, expectedMinedRelays, actualMinedRelays, "TODO_FLAKY: Try re-running with 'go test -v -count=1 -run TestMiner_MinedRelays ./pkg/relayer/miner/...'")
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

	// TODO_TECHDEBT(@red-0ne, #446): Centralize the configuration for the SMT spec.
	relayHashArr := servicetypes.GetHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}
