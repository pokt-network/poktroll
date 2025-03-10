package testrings

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// NewRingClientWithMockDependencies creates a new "real" RingClient with the given
// mock Account and Application queriers supplied as dependencies.
// The queriers are expected to maintain their respective mocked states:
//   - Account querier: the account addresses and public keys
//   - Application querier: the application addresses delegatee gateway addresses
//
// See:
//
//	testutil/testclient/testqueryclients/accquerier.go
//	testutil/testclient/testqueryclients/appquerier.go
//	testutil/testclient/testdelegation/client.go
//
// for methods to create these queriers and maintain their states.
func NewRingClientWithMockDependencies(
	ctx context.Context,
	t *testing.T,
	deps depinject.Config,
) crypto.RingClient {
	t.Helper()

	logger := polylog.Ctx(ctx)
	deps = depinject.Configs(deps, depinject.Supply(logger))

	ringClient, err := rings.NewRingClient(deps)
	require.NoError(t, err)

	return ringClient
}
