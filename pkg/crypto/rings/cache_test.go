package rings

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// TODO(@h5law): Add unit tests for the RingCache
// TODO(@h5law): Add integration tests for the RingCache

func TestRingCache_BuildRing(t *testing.T) {
	rc := createRingCache(t)
	tests := []struct {
		name          string
		appAddress    string
		numDelegatees int
		expectedSize  int
		expectedErr   error
	}{
		{
			name:          "success: un-cached application no delegated gateways",
			appAddress:    sample.AccAddress(),
			numDelegatees: 0,
			expectedSize:  2,
			expectedErr:   nil,
		},
		{
			name:          "success: un-cached application with delegated gateways",
			appAddress:    sample.AccAddress(),
			numDelegatees: 2,
			expectedSize:  3,
			expectedErr:   nil,
		},
	}
	ctx := context.TODO()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// If we expect the application to exist then add it to the test
			// application map with the number of delegated gateways it is
			// supposed to have so it can be retrieved from the mock
			if !errors.As(test.expectedErr, &apptypes.ErrAppNotFound) {
				testqueryclients.AddAddressToApplicationMap(t, test.appAddress, test.numDelegatees)
			}
			// Attempt to retrieve the ring for the address
			ring, err := rc.GetRingForAddress(ctx, test.appAddress)
			if test.expectedErr != nil {
				require.ErrorAs(t, err, &test.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedSize, ring.Size())
		})
	}
}

// createRingCache creates the RingCache using mocked AccountQueryClient and
// ApplicatioQueryClient instances
func createRingCache(t *testing.T) RingCache {
	t.Helper()
	accQuerier := testqueryclients.NewTestAccountQueryClient(t)
	appQuerier := testqueryclients.NewTestApplicationQueryClient(t)
	deps := depinject.Supply(client.AccountQueryClient(accQuerier), client.ApplicationQueryClient(appQuerier))
	rc, err := NewRingCache(deps)
	require.NoError(t, err)
	return rc
}
