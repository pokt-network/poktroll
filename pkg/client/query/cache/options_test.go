package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// testBlock is a minimal client.Block used to drive the committed-blocks observable.
type testBlock struct{ height int64 }

func (b testBlock) Height() int64 { return b.height }
func (b testBlock) Hash() []byte  { return []byte("test_block_hash") }

// countingCache is a Cache which records how many times it has been cleared.
type countingCache struct {
	mu     sync.Mutex
	clears int
}

func (c *countingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clears++
}

func (c *countingCache) numClears() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clears
}

// cacheClearHarness wires a cache's session clear handler to a committed-blocks
// observable that the test drives block by block.
type cacheClearHarness struct {
	t                 *testing.T
	sharedParamsCache client.ParamsCache[sharedtypes.Params]
	cache             *countingCache
	publishCh         chan<- client.Block
}

// newCacheClearHarness registers WithSessionCountCacheClearFn on a counting cache.
//
// The shared params cache starts populated with the default params, matching a
// RelayMiner that has already made at least one query before the block under test.
func newCacheClearHarness(t *testing.T, numSessionsToClearCache uint) *cacheClearHarness {
	t.Helper()

	ctx, cancelCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelCtx)

	blocksObs, publishCh := channel.NewReplayObservable[client.Block](ctx, 1)

	blockClientMock := mockclient.NewMockBlockClient(gomock.NewController(t))
	blockClientMock.EXPECT().
		CommittedBlocksSequence(gomock.Any()).
		Return(blocksObs).
		AnyTimes()

	sharedParamsCache, err := NewParamsCache[sharedtypes.Params](memory.WithTTL(time.Minute))
	require.NoError(t, err)
	sharedParamsCache.Set(sharedtypes.DefaultParams())

	deps := depinject.Supply(blockClientMock, sharedParamsCache, polyzero.NewLogger())

	cache := new(countingCache)
	require.NoError(t, WithSessionCountCacheClearFn(numSessionsToClearCache)(ctx, deps, cache))

	return &cacheClearHarness{
		t:                 t,
		sharedParamsCache: sharedParamsCache,
		cache:             cache,
		publishCh:         publishCh,
	}
}

// commitBlock publishes a block and blocks until the clear handler has observed it.
func (h *cacheClearHarness) commitBlock(height int64) {
	h.t.Helper()

	h.publishCh <- testBlock{height: height}

	// The handler runs on its own goroutine; give it a moment to drain the
	// notification before the test inspects the resulting clear count.
	time.Sleep(20 * time.Millisecond)
}

// requireNumClears asserts the observed clear count, allowing for a late handler.
func (h *cacheClearHarness) requireNumClears(expected int) {
	h.t.Helper()

	require.Eventually(h.t, func() bool {
		return h.cache.numClears() == expected
	}, time.Second, 10*time.Millisecond,
		"expected %d clears, got %d", expected, h.cache.numClears())
}

// TestWithSessionCountCacheClearFn_ClearsWhenSharedParamsCacheIsEmpty is the regression
// test for the fan-out ordering bug.
//
// The shared params cache is itself registered for session clearing, so its handler
// empties it before the other handlers subscribed to the same committed-blocks
// observable read it. Handlers that lost that race saw an empty shared params cache and
// skipped their own clear entirely — leaving the serviceId-keyed service cache serving a
// stale compute_units_per_relay until the process restarted.
//
// A clear handler must therefore still fire when the shared params cache is empty at
// decision time, as long as it has observed shared params at least once before.
func TestWithSessionCountCacheClearFn_ClearsWhenSharedParamsCacheIsEmpty(t *testing.T) {
	// Default params: 10 blocks per session, so session 1 is [1, 10] and session 2 is [11, 20].
	h := newCacheClearHarness(t, 1)

	// Observe a block in session 1. The first block only adopts the in-progress session.
	h.commitBlock(5)
	h.requireNumClears(0)

	// Simulate the shared params cache handler winning the fan-out race and emptying
	// the cache before this cache's handler gets to read it.
	h.sharedParamsCache.Clear()

	// Crossing into session 2 MUST still clear.
	h.commitBlock(11)
	h.requireNumClears(1)
}

// TestWithSessionCountCacheClearFn_ClearsWhenSessionStartBlockIsMissed is the regression
// test for the exact-equality gate.
//
// The clear used to require `currentHeight == currentSessionStartHeight`, so a session
// start block that arrived late or was never delivered skipped that session's clear
// altogether. Tracking the session NUMBER instead means the clear still happens on the
// first block observed within the session.
func TestWithSessionCountCacheClearFn_ClearsWhenSessionStartBlockIsMissed(t *testing.T) {
	h := newCacheClearHarness(t, 1)

	h.commitBlock(5)
	h.requireNumClears(0)

	// Session 2 starts at height 11; deliver height 15 instead, as if 11 were dropped.
	h.commitBlock(15)
	h.requireNumClears(1)
}

// TestWithSessionCountCacheClearFn_ClearsOncePerSession asserts the clear is not
// repeated for every block of a session now that it is no longer pinned to the exact
// session start height.
func TestWithSessionCountCacheClearFn_ClearsOncePerSession(t *testing.T) {
	h := newCacheClearHarness(t, 1)

	h.commitBlock(5)
	h.requireNumClears(0)

	// Session 2: [11, 20]. Only the first observed block clears.
	h.commitBlock(11)
	h.commitBlock(12)
	h.commitBlock(20)
	h.requireNumClears(1)

	// Session 3: [21, 30].
	h.commitBlock(21)
	h.requireNumClears(2)
}

// TestWithSessionCountCacheClearFn_HonorsNumSessionsToClearCache asserts that a clear
// interval greater than one still only clears on matching session numbers.
func TestWithSessionCountCacheClearFn_HonorsNumSessionsToClearCache(t *testing.T) {
	h := newCacheClearHarness(t, 2)

	// Session 1: adopted, no clear.
	h.commitBlock(5)
	h.requireNumClears(0)

	// Session 2 is clearable (2 % 2 == 0).
	h.commitBlock(11)
	h.requireNumClears(1)

	// Session 3 is not.
	h.commitBlock(21)
	h.requireNumClears(1)

	// Session 4 is.
	h.commitBlock(31)
	h.requireNumClears(2)
}

// TestSharedParamsTracker_RetainsLastObservedParams asserts the tracker keeps serving
// the most recently observed params after the underlying cache is emptied, and reports
// that it has none before it ever observes any.
func TestSharedParamsTracker_RetainsLastObservedParams(t *testing.T) {
	sharedParamsCache, err := NewParamsCache[sharedtypes.Params](memory.WithTTL(time.Minute))
	require.NoError(t, err)

	tracker := new(sharedParamsTracker)

	// Nothing observed yet.
	params, found := tracker.observe(sharedParamsCache)
	require.False(t, found)
	require.Nil(t, params)

	expectedParams := sharedtypes.DefaultParams()
	sharedParamsCache.Set(expectedParams)

	params, found = tracker.observe(sharedParamsCache)
	require.True(t, found)
	require.Equal(t, expectedParams.NumBlocksPerSession, params.NumBlocksPerSession)

	// Emptying the cache must not lose the tracked copy.
	sharedParamsCache.Clear()

	params, found = tracker.observe(sharedParamsCache)
	require.True(t, found)
	require.Equal(t, expectedParams.NumBlocksPerSession, params.NumBlocksPerSession)
}
