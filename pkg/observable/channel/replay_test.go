package channel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pocket/internal/testerrors"
	"pocket/pkg/observable/channel"
)

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
		publishCh <- value
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
			// the replay buffer is not full so Last should return values
			// starting from the first published value.
			expectedValues: values[:3], // []int{1, 2, 3},
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
			expectedValues: values[2:], // []int{3, 4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx = context.Background()

			replayObsvbl, publishCh :=
				channel.NewReplayObservable[int](ctx, tt.replayBufferSize)

			for _, value := range values {
				publishCh <- value
				time.Sleep(time.Millisecond)
			}

			actualValues := replayObsvbl.Last(ctx, tt.lastN)
			require.ElementsMatch(t, tt.expectedValues, actualValues)
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
			// Last should block until lastN values have been published.
			// NOTE: this will produce a warning log which can safely be ignored:
			// > WARN: requested replay buffer size 3 is greater than replay buffer
			// > 	   capacity 3; returning entire replay buffer
			lastValuesCh <- replayObsvbl.Last(ctx, lastN)
		}()
		return lastValuesCh
	}

	// Ensure that last blocks when the replay buffer is empty
	select {
	case actualValues := <-getLastValues():
		t.Fatalf(
			"Last should block until at lest 1 value has been published; actualValues: %v",
			actualValues,
		)
	case <-time.After(200 * time.Millisecond):
	}

	// Publish some values (up to splitIdx).
	for _, value := range values[:splitIdx] {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	// Ensure Last works as expected when n <= len(published_values).
	require.ElementsMatch(t, []int{1}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{1, 2}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{1, 2, 3}, replayObsvbl.Last(ctx, 3))

	// Ensure that Last blocks when n > len(published_values) and the replay
	// buffer is not full.
	select {
	case actualValues := <-getLastValues():
		t.Fatalf(
			"Last should block until replayPartialBufferTimeout has elapsed; received values: %v",
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
	case <-time.After(10 * time.Millisecond):
		t.Fatal("timed out waiting for Last to return")
	}

	// Ensure that Last still works as expected when n <= len(published_values).
	require.ElementsMatch(t, []int{1}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{1, 2}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{1, 2, 3}, replayObsvbl.Last(ctx, 3))
	require.ElementsMatch(t, []int{1, 2, 3, 4}, replayObsvbl.Last(ctx, 4))
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5}, replayObsvbl.Last(ctx, 5))
}
