package websockets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// TestGoCancelBridgeOnStop_CancelsOnStopEmit verifies the early-teardown fix: when
// either connection signals stop (disconnect/error via the stop observable), the
// bridge context is canceled immediately instead of lingering until closeHeight.
// Without this, a disconnected websocket client pins the bridge's Run/messageLoop
// goroutines, its block subscription and its observers for the rest of the session.
func TestGoCancelBridgeOnStop_CancelsOnStopEmit(t *testing.T) {
	stopBridgeObservable, stopChan := channel.NewObservable[error]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherReturned := make(chan struct{})
	go func() {
		goCancelBridgeOnStop(ctx, cancel, stopBridgeObservable)
		close(watcherReturned)
	}()

	// Emit a stop signal (as connection.handleError would on disconnect/error).
	// Retry the publish until the context is canceled: the watcher subscribes
	// inside its own goroutine, so an early single publish could race ahead of the
	// subscription and be dropped by the (non-replay) observable.
	require.Eventually(t, func() bool {
		select {
		case stopChan <- errors.New("client disconnected"):
		default:
		}
		return ctx.Err() != nil
	}, 2*time.Second, 10*time.Millisecond, "stop emit did not cancel the bridge context")

	// The watcher must return (it must not leak waiting on the observable).
	select {
	case <-watcherReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher goroutine did not return after canceling the context")
	}
}

// TestGoCancelBridgeOnStop_ExitsOnCtxDone verifies the closeHeight teardown path:
// when Run cancels the bridge context (a committed block reached closeHeight), the
// watcher returns on its own without needing a stop emit, so it never outlives the
// bridge. cancelCtx being called again on this path must be a harmless no-op.
func TestGoCancelBridgeOnStop_ExitsOnCtxDone(t *testing.T) {
	stopBridgeObservable, _ := channel.NewObservable[error]()
	ctx, cancel := context.WithCancel(context.Background())

	watcherReturned := make(chan struct{})
	go func() {
		goCancelBridgeOnStop(ctx, cancel, stopBridgeObservable)
		close(watcherReturned)
	}()

	// Simulate Run canceling the context at closeHeight.
	cancel()

	select {
	case <-watcherReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher goroutine leaked: did not return when the context was canceled")
	}
}
