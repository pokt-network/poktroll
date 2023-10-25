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
		n           = 3
		values      = []int{1, 2, 3, 4, 5}
		errCh       = make(chan error, 1)
		ctx, cancel = context.WithCancel(context.Background())
	)
	t.Cleanup(cancel)

	// NB: intentionally not using NewReplayObservable() to test Replay() directly
	// and to retain a reference to the wrapped observable for testing.
	obsvbl, publishCh := channel.NewObservable[int]()
	replayObsvbl := channel.Replay[int](ctx, n, obsvbl)

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

	// replay observer, should receive the last n values published prior to
	// subscribing followed by subsequently published values
	replayObserver := replayObsvbl.Subscribe(ctx)
	for _, expected := range values[len(values)-n:] {
		select {
		case v := <-replayObserver.Ch():
			require.Equal(t, expected, v)
		case <-time.After(1 * time.Second):
			t.Fatalf("Did not receive expected value %d in time", expected)
		}
	}

	// second replay observer, should receive the same values as the first
	replayObserver2 := replayObsvbl.Subscribe(ctx)
	for _, expected := range values[len(values)-n:] {
		select {
		case v := <-replayObserver2.Ch():
			require.Equal(t, expected, v)
		case <-time.After(1 * time.Second):
			t.Fatalf("Did not receive expected value %d in time", expected)
		}
	}
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

func TestReplayObservable_Last_Blocks_Goroutine(t *testing.T) {
	var (
		n        = 5
		splitIdx = 3
		values   = []int{1, 2, 3, 4, 5}
		ctx      = context.Background()
	)

	replayObsvbl, publishCh := channel.NewReplayObservable[int](ctx, n)

	// Publish values up to splitIdx.
	for _, value := range values[:splitIdx] {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	require.ElementsMatch(t, []int{1}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{1, 2}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{1, 2, 3}, replayObsvbl.Last(ctx, 3))

	// Concurrently call Last with a value greater than the replay buffer size.
	lastValues := make(chan []int, 1)
	go func() {
		// Last should block until n values have been published.
		lastValues <- replayObsvbl.Last(ctx, n)
	}()

	select {
	case actualValues := <-lastValues:
		t.Fatalf(
			"Last should block until the replay buffer is full. Actual values: %v",
			actualValues,
		)
	case <-time.After(10 * time.Millisecond):
	}

	// Publish values after splitIdx.
	for _, value := range values[splitIdx:] {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	select {
	case actualValues := <-lastValues:
		require.ElementsMatch(t, values, actualValues)
	case <-time.After(10 * time.Millisecond):
		t.Fatal("timed out waiting for Last to return")
	}

	require.ElementsMatch(t, []int{1}, replayObsvbl.Last(ctx, 1))
	require.ElementsMatch(t, []int{1, 2}, replayObsvbl.Last(ctx, 2))
	require.ElementsMatch(t, []int{1, 2, 3}, replayObsvbl.Last(ctx, 3))
	require.ElementsMatch(t, []int{1, 2, 3, 4}, replayObsvbl.Last(ctx, 4))
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5}, replayObsvbl.Last(ctx, 5))
}
