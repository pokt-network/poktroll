package testtx

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

type signAndBroadcastFn func(context.Context, cosmostypes.Msg) either.AsyncError

// TODO_CONSIDERATION: functions like these (NewLocalnetXXX) could probably accept
// and return depinject.Config arguments to support shared dependencies.

// NewLocalnetClient creates and returns a new client for use with the LocalNet validator.
func NewLocalnetClient(t *testing.T, opts ...client.TxClientOption) client.TxClient {
	t.Helper()

	ctx := context.Background()
	txCtx := NewLocalnetContext(t)
	eventsQueryClient := testeventsquery.NewLocalnetClient(t)
	blockClient := testblock.NewLocalnetClient(ctx, t)

	deps := depinject.Supply(
		txCtx,
		eventsQueryClient,
		blockClient,
	)

	txClient, err := tx.NewTxClient(ctx, deps, opts...)
	require.NoError(t, err)

	return txClient
}

// NewOneTimeDelayedSignAndBroadcastTxClient constructs a mock TxClient with the
// expectation to perform a SignAndBroadcast operation with a specified delay.
func NewOneTimeDelayedSignAndBroadcastTxClient(
	t *testing.T,
	ctx context.Context,
	delay time.Duration,
) *mockclient.MockTxClient {
	t.Helper()

	signAndBroadcast := newSignAndBroadcastSucceedsDelayed(delay)
	return NewOneTimeSignAndBroadcastTxClient(t, ctx, signAndBroadcast)
}

// NewOneTimeSignAndBroadcastTxClient constructs a mock TxClient with the
// expectation to perform a SignAndBroadcast operation, which will call and receive
// the return from the given signAndBroadcast function.
func NewOneTimeSignAndBroadcastTxClient(
	t *testing.T,
	ctx context.Context,
	signAndBroadcast signAndBroadcastFn,
) *mockclient.MockTxClient {
	t.Helper()

	ctrl := gomock.NewController(t)

	txClient := mockclient.NewMockTxClient(ctrl)
	txClient.EXPECT().SignAndBroadcast(
		gomock.Eq(ctx),
		gomock.Any(),
	).DoAndReturn(signAndBroadcast).Times(1)

	return txClient
}

// newSignAndBroadcastSucceedsDelayed returns a signAndBroadcastFn that succeeds
// after the given delay.
func newSignAndBroadcastSucceedsDelayed(delay time.Duration) signAndBroadcastFn {
	return func(ctx context.Context, msg cosmostypes.Msg) either.AsyncError {
		errCh := make(chan error)

		go func() {
			time.Sleep(delay)
			close(errCh)
		}()

		return either.AsyncErr(errCh)
	}
}
