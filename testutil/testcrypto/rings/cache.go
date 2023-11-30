package testrings

import (
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// NewRingCacheWithMockQueriers creates a new "real" RingCache with the given
// mock Account and Application queriers supplied as dependencies.
// The queriers are expected to maintain their respective mocked states:
//   - Account querier: the account addresses and public keys
//   - Application querier: the application addresses delegatee gateway addresses
//
// See:
//
//	testutil/testclient/testqueryclients/accquerier.go
//	testutil/testclient/testqueryclients/appquerier.go
//
// for methods to create these queriers and maintain their states.
func NewRingCacheWithMockQueriers(
	t *testing.T,
	accQuerier *mockclient.MockAccountQueryClient,
	appQuerier *mockclient.MockApplicationQueryClient,
) crypto.RingCache {
	t.Helper()

	// Create the dependency injector with the mock queriers
	deps := depinject.Supply(accQuerier, appQuerier)

	ringCache, err := rings.NewRingCache(deps)
	require.NoError(t, err)

	return ringCache
}
