package channel_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testerrors"
)

func TestReplayObservable_Overflow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	replayObs, replayPublishCh := channel.NewReplayObservable[int](context.Background(), 6)

	// Ensure that the replay observable can handle synchronous publishing
	replayPublishCh <- 0
	replayPublishCh <- 1
	replayPublishCh <- 2

	// Publish values asynchronously at different intervals
	go func() {
		time.Sleep(time.Millisecond)
		replayPublishCh <- 3
	}()

	go func() {
		time.Sleep(20 * time.Millisecond)
		replayPublishCh <- 4
	}()

	go func() {
		time.Sleep(40 * time.Millisecond)
		replayPublishCh <- 5
	}()

	// Assert that calling last synchronously returns the synchronously published values.
	actualValues := replayObs.Last(ctx, 3)
	require.ElementsMatch(t, []int{2, 1, 0}, actualValues)

	// Assert that the items returned by Last are the expected ones according to
	// when they were published.

	time.Sleep(10 * time.Millisecond)
	actualValues = replayObs.Last(ctx, 3)
	require.ElementsMatch(t, []int{3, 2, 1}, actualValues)

	time.Sleep(20 * time.Millisecond)
	actualValues = replayObs.Last(ctx, 3)
	require.ElementsMatch(t, []int{4, 3, 2}, actualValues)

	time.Sleep(20 * time.Millisecond)
	actualValues = replayObs.Last(ctx, 3)
	require.ElementsMatch(t, []int{5, 4, 3}, actualValues)
}

func TestReplayObservable(t *testing.T) {
	var (
		replayBufferSize = 3
		values           = []int{1, 2, 3, 4, 5}
		// the replay buffer is full and has shifted out values with index <
		// len(values)-replayBufferSize so Last should return values starting
		// from there.
		expectedValues = values[len(values)-replayBufferSize:]
		errCh          = make(chan error, 1)
		ctx, cancel    = context.WithCancel(context.Background())
	)
	t.Cleanup(cancel)

	// NB: intentionally not using NewReplayObservable() to test ToReplayObservable() directly
	// and to retain a reference to the wrapped observable for testing.
	obsvbl, publishCh := channel.NewObservable[int]()
	replayObsvbl := channel.ToReplayObservable[int](ctx, replayBufferSize, obsvbl)

	// vanilla observer, should be able to receive all values published after subscribing
	observer := obsvbl.Subscribe(ctx)
	go func() {
		for _, expected := range values {
			select {
			case v := <-observer.Ch():
				if !assert.Equal(t, expected, v) {
					errCh <- testerrors.ErrAsync
					return
				}
			case <-time.After(1 * time.Second):
				t.Errorf("Did not receive expected value %d in time", expected)
				errCh <- testerrors.ErrAsync
				return
			}
		}
	}()

	// send all values to the observable's publish channel
	for _, value := range values {
		time.Sleep(10 * time.Microsecond)
		publishCh <- value
		time.Sleep(10 * time.Microsecond)
	}

	// allow some time for values to be buffered by the replay observable
	time.Sleep(time.Millisecond)

	// replay observer, should receive the last lastN values published prior to
	// subscribing followed by subsequently published values
	replayObserver := replayObsvbl.Subscribe(ctx)

	// Collect values from replayObserver.
	var actualValues []int
	for _, expected := range expectedValues {
		select {
		case v := <-replayObserver.Ch():
			actualValues = append(actualValues, v)
		case <-time.After(1 * time.Second):
			t.Fatalf("Did not receive expected value %d in time", expected)
		}
	}

	require.EqualValues(t, expectedValues, actualValues)

	// Second replay observer, should receive the same values as the first
	// even though it subscribed after all values were published and the
	// values were already replayed by the first.
	replayObserver2 := replayObsvbl.Subscribe(ctx)

	// Collect values from replayObserver2.
	var actualValues2 []int
	for _, expected := range expectedValues {
		select {
		case v := <-replayObserver2.Ch():
			actualValues2 = append(actualValues2, v)
		case <-time.After(1 * time.Second):
			t.Fatalf("Did not receive expected value %d in time", expected)
		}
	}

	require.EqualValues(t, expectedValues, actualValues)
}

func TestReplayObservable_Last_Full_ReplayBuffer(t *testing.T) {
	values := []int{1, 2, 3, 4, 5}
	expectedValues := values
	// Reverse the expected values to have the most recent values first.
	slices.Reverse(expectedValues)

	tests := []struct {
		name             string
		replayBufferSize int
		// lastN is the number of values to return from the replay buffer
		lastN          int
		expectedValues []int
	}{
		{
			name:             "n < replayBufferSize",
			replayBufferSize: 5,
			lastN:            3,
			// the replay buffer has enough values to return to Last, it should return
			// the last n values in the replay buffer.
			expectedValues: values[2:], // []int{5, 4, 3},
		},
		{
			name:             "n = replayBufferSize",
			replayBufferSize: 5,
			lastN:            5,
			expectedValues:   values,
		},
		{
			name:             "n > replayBufferSize",
			replayBufferSize: 3,
			lastN:            5,
			// the replay buffer is full so Last should return values starting
			// from lastN - replayBufferSize.
			expectedValues: values[2:], // []int{5, 4, 3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()

			replayObsvbl, publishCh :=
				channel.NewReplayObservable[int](ctx, test.replayBufferSize)

			for _, value := range values {
				publishCh <- value
				time.Sleep(time.Millisecond)
			}

			actualValues := replayObsvbl.Last(ctx, test.lastN)
			require.ElementsMatch(t, test.expectedValues, actualValues)
		})
	}
}

func TestReplayObservable_Last_Blocks_And_Times_Out(t *testing.T) {
	var (
		replayBufferSize = 5
		lastN            = 5
		// splitIdx is the index at which this test splits the set of values.
		// The two groups of values are published at different points in the
		// test to test the behavior of Last under different conditions.
		splitIdx = 3
		values   = []int{1, 2, 3, 4, 5}
		ctx      = context.Background()
	)

	replayObsvbl, publishCh := channel.NewReplayObservable[int](ctx, replayBufferSize)

	// getLastValues is a helper function which returns a channel that will
	// receive the result of a call to Last, the method under test.
	getLastValues := func() chan []int {
		lastValuesCh := make(chan []int, 1)
		go func() {
			// The replay observable's Last method does not timeout if there is less
			// than lastN values in the replay buffer.
			// Add a timeout to ensure that Last doesn't block indefinitely and return
			// whatever values are available in the replay buffer.
			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			// Last should block until lastN values have been published or the timeout
			// specified above is reached.
			// NOTE: this will produce a warning log which can safely be ignored:
			// > WARN: requested replay buffer size 3 is greater than replay buffer
			// > 	   capacity 3; returning entire replay buffer
			lastValuesCh <- replayObsvbl.Last(ctx, lastN)
			cancel()
		}()
		return lastValuesCh
	}

	// Ensure that Last blocks when the replay buffer is empty
	select {
	case actualValues := <-getLastValues():
		t.Fatalf(
			"Last should block until it gets %d values; actualValues: %v",
			lastN,
			actualValues,
		)
	case <-time.After(50 * time.Millisecond):
	}

	// Publish some values (up to splitIdx).
	for _, value := range values[:splitIdx] {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	// Ensure Last works as expected when n <= len(published_values).
	require.ElementsMatch(t, []int{3}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{3, 2}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{3, 2, 1}, replayObsvbl.Last(ctx, 3))

	// Ensure that Last blocks when n > len(published_values) and the replay
	// buffer is not full.
	select {
	case actualValues := <-getLastValues():
		t.Fatalf(
			"Last should block until %d items are published; received values: %v",
			lastN,
			actualValues,
		)
	default:
		t.Log("OK: Last is blocking, as expected")
	}

	// Ensure that Last returns the correct values when n > len(published_values)
	// and the replay buffer is not full.
	select {
	case actualValues := <-getLastValues():
		require.ElementsMatch(t, values[:splitIdx], actualValues)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for Last to return")
	}

	// Publish the rest of the values (from splitIdx on).
	for _, value := range values[splitIdx:] {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	// Ensure that Last doesn't block when n = len(published_values) and the
	// replay buffer is full.
	select {
	case actualValues := <-getLastValues():
		require.Len(t, actualValues, lastN)
		require.ElementsMatch(t, values, actualValues)
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timed out waiting for Last to return")
	}

	// Ensure that Last still works as expected when n <= len(published_values).
	// The values are ordered from most recent to least recent.
	require.ElementsMatch(t, []int{5}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{5, 4}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{5, 4, 3}, replayObsvbl.Last(ctx, 3))
	require.ElementsMatch(t, []int{5, 4, 3, 2}, replayObsvbl.Last(ctx, 4))
	require.ElementsMatch(t, []int{5, 4, 3, 2, 1}, replayObsvbl.Last(ctx, 5))
}

func TestReplayObservable_SubscribeFromLatestBufferedOffset(t *testing.T) {
	receiveTimeout := 100 * time.Millisecond
	values := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	tests := []struct {
		name             string
		replayBufferSize int
		endOffset        int
		expectedValues   []int
	}{
		{
			name:             "endOffset = replayBufferSize",
			replayBufferSize: 8,
			endOffset:        8,
			expectedValues:   values[2:], // []int{2, 3, 4, 5, ...},
		},
		{
			name:             "endOffset < replayBufferSize",
			replayBufferSize: 10,
			endOffset:        2,
			expectedValues:   values[8:], // []int{8, 9},
		},
		{
			name:             "endOffset > replayBufferSize",
			replayBufferSize: 8,
			endOffset:        10,
			expectedValues:   values[2:],
		},
		{
			name:             "replayBufferSize < eldOffset < numBufferedValues ",
			replayBufferSize: 20,
			endOffset:        15,
			expectedValues:   values,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ctx = context.Background()

			replayObsvbl, publishCh :=
				channel.NewReplayObservable[int](ctx, test.replayBufferSize)

			for _, value := range values {
				publishCh <- value
				time.Sleep(time.Millisecond)
			}

			observer := replayObsvbl.SubscribeFromLatestBufferedOffset(ctx, test.endOffset)
			// Assumes all values will be received within receiveTimeout.
			actualValues := accumulateValues(observer, receiveTimeout)
			require.EqualValues(t, test.expectedValues, actualValues)
		})
	}
}

func accumulateValues[V any](
	observer observable.Observer[V],
	timeout time.Duration,
) (values []V) {
	for {
		select {
		case value, ok := <-observer.Ch():
			if !ok {
				return
			}

			values = append(values, value)
			continue
		case <-time.After(timeout):
			return
		}
	}
}
