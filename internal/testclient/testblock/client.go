package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/internal/testclient"
	"github.com/pokt-network/poktroll/internal/testclient/testeventsquery"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
)

func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	bClient, err := block.NewBlockClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)

	return bClient
}
