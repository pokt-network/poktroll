package websockets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// TestConnection_handleError_recoversOnClosedStopChan reproduces the production
// crash: a stopChan sender (handleError, reachable from connLoop, pingLoop, and
// the bridge's messageLoop) racing the bridge's teardown close of stopChan.
//
// A send on a closed channel panics, and before the recover backstop this
// crashed the entire RelayMiner process ("panic: send on closed channel" at
// connection.handleError, crash-looping under websocket-backend churn). The
// backstop must turn the late send into a dropped, logged signal instead.
func TestConnection_handleError_recoversOnClosedStopChan(t *testing.T) {
	logger := polyzero.NewLogger()

	stopCh := make(chan error, 1)
	close(stopCh) // simulate the bridge having already closed stopChan on teardown

	c := &connection{
		ctx:               context.Background(),
		logger:            logger,
		handleErrorLogger: logger,
		serviceID:         "test-svc",
		stopChan:          stopCh,
	}

	require.NotPanics(t, func() {
		c.handleError(errors.New("late error during bridge shutdown"))
	})
}

// TestConnection_handleError_deliversOnOpenStopChan verifies the normal path is
// unchanged: when stopChan is open, the error is delivered to it.
func TestConnection_handleError_deliversOnOpenStopChan(t *testing.T) {
	logger := polyzero.NewLogger()

	stopCh := make(chan error, 1)

	c := &connection{
		ctx:               context.Background(),
		logger:            logger,
		handleErrorLogger: logger,
		serviceID:         "test-svc",
		stopChan:          stopCh,
	}

	wantErr := errors.New("boom")
	c.handleError(wantErr)

	select {
	case got := <-stopCh:
		require.ErrorIs(t, got, wantErr)
	default:
		t.Fatal("expected handleError to deliver the error on the open stopChan")
	}
}
