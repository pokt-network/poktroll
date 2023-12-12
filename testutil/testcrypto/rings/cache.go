package testrings

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
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
	accQuerier *mockclient.MockAccountQueryClient,
	appQuerier *mockclient.MockApplicationQueryClient,
	delegationClient *mockclient.MockDelegationClient,
) crypto.RingCache {
	t.Helper()

	// Create the dependency injector with the mock queriers
	logger := polylog.Ctx(ctx)
	deps := depinject.Supply(logger, accQuerier, appQuerier, delegationClient)

	ringCache, err := rings.NewRingCache(deps)
	require.NoError(t, err)

	return ringCache
}
