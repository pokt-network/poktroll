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

// NewRingCacheWithMockDependencies creates a new "real" RingCache with the given
// mock Account and Application queriers supplied as dependencies. A Delegation
// client is required as a dependency and depending on how it is used will
// require a different function to generate the delegations client.
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
func NewRingCacheWithMockDependencies(
	ctx context.Context,
	t *testing.T,
	deps depinject.Config,
) crypto.RingCache {
	t.Helper()

	logger := polylog.Ctx(ctx)
	deps = depinject.Configs(deps, depinject.Supply(logger))

	ringCache, err := rings.NewRingCache(deps)
	require.NoError(t, err)

	return ringCache
}
