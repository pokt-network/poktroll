package channel_test

import (
	"context"
	"crypto/sha256"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// cpuBoundHash mimics the miner's per-relay marshal+hash work: a CPU-bound,
// pure, per-item transform. Iterated to make the cost dominate channel overhead.
func cpuBoundHash(_ context.Context, n int) ([32]byte, bool) {
	buf := []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
	h := sha256.Sum256(buf)
	for i := 0; i < 200; i++ {
		h = sha256.Sum256(h[:])
	}
	return h, false
}

func benchmarkMapVariant(b *testing.B, parallel bool, numWorkers int) {
	b.Helper()
	const numItems = 2_000

	for n := 0; n < b.N; n++ {
		ctx, cancel := context.WithCancel(context.Background())
		srcObservable, srcPublishCh := channel.NewObservable[int]()

		if parallel {
			dst := channel.MapParallel(ctx, srcObservable, cpuBoundHash, numWorkers)
			drainHashes(srcPublishCh, dst.Subscribe(ctx).Ch(), numItems)
		} else {
			dst := channel.Map(ctx, srcObservable, cpuBoundHash)
			drainHashes(srcPublishCh, dst.Subscribe(ctx).Ch(), numItems)
		}
		cancel()
	}
}

// drainHashes publishes numItems integers and blocks until every transformed
// value has been received and the output channel is closed.
func drainHashes(publishCh chan<- int, out <-chan [32]byte, numItems int) {
	done := make(chan struct{})
	go func() {
		for range out {
		}
		close(done)
	}()
	for i := 0; i < numItems; i++ {
		publishCh <- i
	}
	close(publishCh)
	<-done
}

// BenchmarkMap_Serial is the single-goroutine baseline (current pipeline shape).
func BenchmarkMap_Serial(b *testing.B) { benchmarkMapVariant(b, false, 0) }

// BenchmarkMapParallel_8 fans the same CPU-bound work across 8 workers.
func BenchmarkMapParallel_8(b *testing.B) { benchmarkMapVariant(b, true, 8) }

// TestMapParallel_ProcessesAllItems verifies that MapParallel transforms every
// input item exactly once across multiple workers, even though output order is
// not preserved.
func TestMapParallel_ProcessesAllItems(t *testing.T) {
	const (
		numItems   = 1_000
		numWorkers = 8
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srcObservable, srcPublishCh := channel.NewObservable[int]()

	// Double each value on a pool of workers.
	dstObservable := channel.MapParallel(
		ctx,
		srcObservable,
		func(_ context.Context, n int) (int, bool) {
			return n * 2, false
		},
		numWorkers,
	)
	dstObserver := dstObservable.Subscribe(ctx)

	var (
		mu      sync.Mutex
		results = make(map[int]struct{})
		done    = make(chan struct{})
	)
	go func() {
		for v := range dstObserver.Ch() {
			mu.Lock()
			results[v] = struct{}{}
			mu.Unlock()
		}
		close(done)
	}()

	for i := 0; i < numItems; i++ {
		srcPublishCh <- i
	}
	// Closing the source publish channel closes the source observer channels,
	// which drains the workers and (via wg.Wait) closes the destination producer.
	close(srcPublishCh)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for MapParallel output to drain")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, results, numItems, "every input must be transformed exactly once")
	for i := 0; i < numItems; i++ {
		_, ok := results[i*2]
		require.Truef(t, ok, "missing transformed value for input %d (expected %d)", i, i*2)
	}
}

// TestMapParallel_Skip verifies that the skip return value drops items, and that
// MapParallel still closes cleanly when some items are skipped.
func TestMapParallel_Skip(t *testing.T) {
	const numItems = 100

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srcObservable, srcPublishCh := channel.NewObservable[int]()

	// Keep only even numbers.
	dstObservable := channel.MapParallel(
		ctx,
		srcObservable,
		func(_ context.Context, n int) (int, bool) {
			return n, n%2 != 0 // skip odds
		},
		4,
	)
	dstObserver := dstObservable.Subscribe(ctx)

	var (
		mu    sync.Mutex
		count int
		done  = make(chan struct{})
	)
	go func() {
		for range dstObserver.Ch() {
			mu.Lock()
			count++
			mu.Unlock()
		}
		close(done)
	}()

	for i := 0; i < numItems; i++ {
		srcPublishCh <- i
	}
	close(srcPublishCh)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for MapParallel output to drain")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, numItems/2, count, "only even items should be emitted")
}

// TestMapParallel_AutoWorkers verifies that a non-positive worker count is
// accepted (falls back to GOMAXPROCS) and still processes all items.
func TestMapParallel_AutoWorkers(t *testing.T) {
	const numItems = 200

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srcObservable, srcPublishCh := channel.NewObservable[int]()

	dstObservable := channel.MapParallel(
		ctx,
		srcObservable,
		func(_ context.Context, n int) (int, bool) { return n, false },
		0, // auto
	)
	dstObserver := dstObservable.Subscribe(ctx)

	var (
		mu    sync.Mutex
		count int
		done  = make(chan struct{})
	)
	go func() {
		for range dstObserver.Ch() {
			mu.Lock()
			count++
			mu.Unlock()
		}
		close(done)
	}()

	for i := 0; i < numItems; i++ {
		srcPublishCh <- i
	}
	close(srcPublishCh)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for MapParallel output to drain")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, numItems, count)
}
