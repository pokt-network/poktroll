//go:build integration

package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"pocket/internal/testclient/testquery"
)

const (
	msgReceiveTimeout = 5 * time.Second
	msgReceiveLimit   = 2
)

func TestQueryClient_SubscribeWithQuery_NewBlock(t *testing.T) {
	ctx := context.Background()

	queryClient := testquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	msgCh, errCh := make(chan []byte), make(chan error, 1)
	msgHandler := func(ctx context.Context, msg []byte) error {
		msgCh <- msg
		return nil
	}
	queryClient.SubscribeWithQuery(ctx, "tm.event='NewBlock'", msgHandler)

	var msgCounter int
	go func() {
		for range msgCh {
			msgCounter++

			if msgCounter >= msgReceiveLimit {
				errCh <- nil
				return
			}
		}
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, msgReceiveLimit, msgCounter)
	case <-time.After(msgReceiveTimeout):
		t.Fatalf(
			"timed out waiting for block subscription; expected %d blocks, got %d",
			msgReceiveLimit, msgCounter,
		)
	}
}
