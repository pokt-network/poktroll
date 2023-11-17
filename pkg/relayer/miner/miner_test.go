package miner_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"hash"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const testDifficulty = 2

var (
	// marshaledMinableRelaysHex are the hex encoded strings of serialized relays
	// which have been pre-mined to difficulty 2 by populating the signature with
	// random bytes. It is intended for use in tests.
	marshaledMinableRelaysHex = []string{
		"0a140a12121084e353443a333908e9f883c66914a0fb",
		"0a140a121210044c754c27cf71b6b0b4da4db65a9519",
		"0a140a121210e7ae4950528a2f452bdcb753119acb84",
		"0a140a121210d03a97e53b83db84550ee0a4d10f9182",
		"0a140a121210b16c079b8317f0d60435e6a3d4ba185f",
	}

	// marshaledUnminableRelaysHex are the hex encoded strings of serialized relays
	// which have been pre-mined to **exclude** relays with difficulty 2 (or greater).
	// Like marshaledMinableRelaysHex, this is done by populating the signature with
	// random bytes. It is intended for use in tests.
	marshaledUnminableRelaysHex = []string{
		"0a140a121210ec621c4e66d50a7fbd9cab8055f33340",
		"0a140a121210d5b6f79c4a0a5a61a71082d6ffcbba06",
		"0a140a121210963c95b5af267a1b04bce4b6aa1a6684",
		"0a140a121210a303a033c99f91841051b3fcc0984822",
		"0a140a121210d9580e5db33ae495e7805fa0ee0f13ef",
	}
)

func TestMiner_MinedRelays(t *testing.T) {
	var (
		minedRelayCounter                     = 0
		ctx                                   = context.Background()
		actualMinedRelaysMu                   sync.Mutex
		actualMinedRelays                     []*relayer.MinedRelay
		mockRelaysObs, relaysFixturePublishCh = channel.NewObservable[*servicetypes.Relay]()
		expectedMinedRelays                   = unmarshalHexMinedRelays(
			t, marshaledMinableRelaysHex,
			miner.DefaultRelayHasher,
		)
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
	require.EqualValues(t, expectedMinedRelays, actualMinedRelays)
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
	newHasher func() hash.Hash,
) (relays []*relayer.MinedRelay) {
	t.Helper()

	for _, marshalledRelayHex := range marshalledHexMinedRelays {
		relays = append(relays, unmarshalHexMinedRelay(t, marshalledRelayHex, newHasher))
	}
	return relays
}

func unmarshalHexMinedRelay(
	t *testing.T,
	marshalledHexMinedRelay string,
	newHasher func() hash.Hash,
) *relayer.MinedRelay {
	t.Helper()

	relayBz, err := hex.DecodeString(marshalledHexMinedRelay)
	require.NoError(t, err)

	var relay servicetypes.Relay
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	relayHashBz := hashRelay(t, newHasher, relayBz)

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHashBz,
	}
}

func hashRelay(t *testing.T, newHasher func() hash.Hash, relayBz []byte) []byte {
	t.Helper()

	hasher := newHasher()
	_, err := hasher.Write(relayBz)
	require.NoError(t, err)
	return hasher.Sum(nil)
}

func TestFixtureGeneration_MineMockRelays(t *testing.T) {
	t.Skip("this test is intended to be run manually as a utility to generate relay fixtures for testing")

	ctx := context.Background()

	minedRelaysObs := mineRelayFixturesForDifficulty(
		t,
		16, // number of random bytes provided for relay generation
		testDifficulty,
		5, // number of required relays (passing testDifficulty)
		miner.DefaultRelayHasher,
	)
	minedRelaysObserver := minedRelaysObs.Subscribe(ctx)

	for minedRelay := range minedRelaysObserver.Ch() {
		minedRelayBz, err := minedRelay.Marshal()
		require.NoError(t, err)

		t.Logf("%x", minedRelayBz)
	}
}

// mineRelayFixturesForDifficulty is a single-threaded utility for generating
// relay fixtures for testing. It returns an observable of mined relays which
// are published as they are mined, where difficulty is the difficulty threshold
// in bytes. Each relay fixture is populated with a randomized signature of
// randLength length. Relay fixtures are then hashed using the hasher returned
// from newHasher and checked against the difficulty threshold. If the relay
// meets the difficulty threshold, it is published to the returned observable.
// It stops mining & publishing once limit number of relay fixtures have been
// published.
// It is not intended to be used **in/by** any tests but rather is persisted to
// aid in re-generation of relay fixtures should the test requirements change.
func mineRelayFixturesForDifficulty(
	t *testing.T,
	randLength int,
	difficulty int,
	limit int,
	newHasher func() hash.Hash,
) observable.Observable[*relayer.MinedRelay] {
	t.Helper()

	var (
		mined                      = 0
		attempted                  = 0
		randBzObs, randBzPublishCh = channel.NewObservable[*relayer.MinedRelay]()
	)

	go func() {
		for {
			if mined >= limit {
				break
			}

			randBz := make([]byte, randLength)
			n, err := rand.Read(randBz)
			require.NoError(t, err)
			require.Equal(t, randLength, n)

			// Populate a relay with the minimally sufficient randomized data.
			relay := servicetypes.Relay{
				Req: &servicetypes.RelayRequest{
					Meta: &servicetypes.RelayRequestMetadata{
						Signature: randBz,
					},
					Payload: nil,
				},
				Res: nil,
			}

			// TODO_BLOCKER: use canonical codec.
			relayBz, err := relay.Marshal()
			require.NoError(t, err)

			// Hash relay bytes
			relayHash := hashRelay(t, newHasher, relayBz)

			// TODO_TECHDEBT(#192): react to refactoring of protocol package.
			// Check difficulty & publish.
			if !protocol.BytesDifficultyGreaterThan(relayHash, difficulty) {
				randBzPublishCh <- &relayer.MinedRelay{
					Relay: relay,
					Bytes: relayBz,
					Hash:  relayHash,
				}
				mined++
			}
			attempted++

			// Log occasionally; signal liveness/progress.
			if attempted%100000 == 0 {
				t.Logf("attempted: %d, mined: %d", attempted, mined)
			}
		}
		close(randBzPublishCh)
	}()

	return randBzObs
}
