package config_test

import (
	"context"
	"net/url"
	"testing"

	"cosmossdk.io/depinject"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// TestNewSupplyCometClientFn_SuppliesCorrectType is a regression test for issue #1815.
//
// Background:
// PR #1536 (June 2025) replaced EventsQueryClient with CometBFT's native client but
// introduced a subtle bug: NewSupplyCometClientFn supplied the concrete type (*rpchttp.HTTP)
// to depinject instead of the interface type (cometclient.Client).
//
// While newer versions of depinject (v1.2.0+) may handle this automatically in some contexts,
// users reported failures when running migration commands:
//   "can't resolve type github.com/cometbft/cometbft/rpc/client/client.Client"
//
// This test ensures that:
// 1. The CometBFT client is properly supplied to the dependency injection container
// 2. Consumers can successfully inject and use the client
// 3. The specific use case from migration commands works (NewEventsReplayClient)
//
// If this test fails, it likely means NewSupplyCometClientFn is not properly casting
// the concrete type to the interface when calling depinject.Supply().
func TestNewSupplyCometClientFn_SuppliesCorrectType(t *testing.T) {
	ctx := context.Background()

	// Setup: Create RPC URL and supply logger
	queryNodeRPCURL, err := url.Parse("http://localhost:26657")
	require.NoError(t, err)

	logger := polylog.Ctx(ctx)
	deps := depinject.Configs(depinject.Supply(logger))

	// Act: Supply the CometBFT client using the production function
	supplierFn := config.NewSupplyCometClientFn(queryNodeRPCURL)
	deps, err = supplierFn(ctx, deps, nil)
	require.NoError(t, err, "Failed to supply CometBFT client")

	// Assert 1: Verify we can inject the client as the interface type
	var cometClient cometclient.Client
	err = depinject.Inject(deps, &cometClient)
	require.NoError(t, err, "Failed to inject cometclient.Client - check if NewSupplyCometClientFn properly casts to interface type")
	require.NotNil(t, cometClient, "Injected cometClient should not be nil")

	// Assert 2: Verify the migration command use case works
	// This is what actually failed for users when running claim-supplier, etc.
	eventsClient, err := events.NewEventsReplayClient[any](
		ctx,
		deps,
		"tm.event='NewBlock'",
		func(event *coretypes.ResultEvent) (any, error) { return nil, nil },
		1,
	)
	require.NoError(t, err, "Failed to create EventsReplayClient - this is what broke migration commands in issue #1815")
	require.NotNil(t, eventsClient, "EventsReplayClient should not be nil")
}
