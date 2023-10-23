package channel_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"pocket/pkg/observable/channel"
)

func TestReplayObservable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	n := 3
	values := []int{1, 2, 3, 4, 5}

	obsvbl, publishCh := channel.NewObservable[int]()
	replayObsvbl := channel.Replay[int](ctx, n, obsvbl)

	// vanilla observer, should be able to receive all values published after subscribing
	observer := obsvbl.Subscribe(ctx)
	go func() {
		for _, expected := range values {
			select {
			case v := <-observer.Ch():
				if v != expected {
					t.Errorf("Expected value %d, but got %d", expected, v)
				}
			case <-time.After(1 * time.Second):
				t.Errorf("Did not receive expected value %d in time", expected)
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
			if v != expected {
				t.Errorf("Expected value %d, but got %d", expected, v)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Did not receive expected value %d in time", expected)
		}
	}

	// second replay observer, should receive the same values as the first
	replayObserver2 := replayObsvbl.Subscribe(ctx)
	for _, expected := range values[len(values)-n:] {
		select {
		case v := <-replayObserver2.Ch():
			if v != expected {
				t.Errorf("Expected value %d, but got %d", expected, v)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Did not receive expected value %d in time", expected)
		}
	}
}

func TestReplayObservable_Next(t *testing.T) {
	ctx := context.Background()
	n := 3
	values := []int{1, 2, 3, 4, 5}

	obsvbl, publishCh := channel.NewObservable[int]()
	replayObsvbl := channel.Replay[int](ctx, n, obsvbl)

	for _, value := range values {
		publishCh <- value
		time.Sleep(time.Millisecond)
	}

	require.Equal(t, 3, replayObsvbl.Next(ctx))
	require.Equal(t, 3, replayObsvbl.Next(ctx))
}
