package rings

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// account is a struct used to define an address public key pairing
type account struct {
	address string
	pubKey  cryptotypes.PubKey
}

// newAccount creates a new account for testing purposes on the desired curve
func newAccount(curve string) account {
	var addr string
	var pubkey cryptotypes.PubKey
	if curve == "ed25519" {
		addr, pubkey = sample.AccAddressAndPubKeyEdd2519()
	} else if curve == "secp256k1" {
		addr, pubkey = sample.AccAddressAndPubKey()
	}
	return account{
		address: addr,
		pubKey:  pubkey,
	}
}

func TestRingCache_BuildRing_Uncached(t *testing.T) {
	rc := createRingCache(t)
	tests := []struct {
		name              string
		appAccount        account
		delegateeAccounts []account
		expectedSize      int
		expectedErr       error
	}{
		{
			name:              "success: un-cached application no delegated gateways",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{},
			expectedSize:      2,
			expectedErr:       nil,
		},
		{
			name:              "success: un-cached application with delegated gateways",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("secp256k1"), newAccount("secp256k1")},
			expectedSize:      3,
			expectedErr:       nil,
		},
		{
			name:              "failure: app pubkey uses wrong curve",
			appAccount:        newAccount("ed25519"),
			delegateeAccounts: []account{newAccount("secp256k1"), newAccount("secp256k1")},
			expectedSize:      0,
			expectedErr:       ErrRingsWrongCurve,
		},
		{
			name:              "failure: gateway pubkey uses wrong curve",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("ed25519"), newAccount("ed25519")},
			expectedSize:      0,
			expectedErr:       ErrRingsWrongCurve,
		},
		{
			name:              "failure: application not found",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("secp256k1")},
			expectedSize:      0,
			expectedErr:       apptypes.ErrAppNotFound,
		},
	}
	ctx := context.TODO()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// If we expect the application to exist then add it to the test
			// application map with the number of delegated gateways it is
			// supposed to have so it can be retrieved from the mock
			if !errors.As(test.expectedErr, &apptypes.ErrAppNotFound) {
				accMap := make(map[string]cryptotypes.PubKey)
				for _, acc := range test.delegateeAccounts {
					accMap[acc.address] = acc.pubKey
				}
				// add the application's account and the accounts of all its
				// delegated gateways to the testing state
				testqueryclients.AddAddressToApplicationMap(t, test.appAccount.address, test.appAccount.pubKey, accMap)
			}
			// Attempt to retrieve the ring for the address
			ring, err := rc.GetRingForAddress(ctx, test.appAccount.address)
			if test.expectedErr != nil {
				require.ErrorAs(t, err, &test.expectedErr)
				return
			}
			require.NoError(t, err)
			// Ensure the ring is the correct size.
			require.Equal(t, test.expectedSize, ring.Size())
		})
	}
}

func TestRingCache_BuildRing_Cached(t *testing.T) {
	rc := createRingCache(t)
	tests := []struct {
		name         string
		appAccount   account
		expectedSize int
		expectedErr  error
	}{
		{
			name:         "success: cached application no delegated gateways",
			appAccount:   newAccount("secp256k1"),
			expectedSize: 2,
			expectedErr:  nil,
		},
		{
			name:         "success: cached application with delegated gateways",
			appAccount:   newAccount("secp256k1"),
			expectedSize: 3,
			expectedErr:  nil,
		},
	}
	ctx := context.TODO()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			accMap := make(map[string]cryptotypes.PubKey)
			// if the test expects a ring > 2 we have delegated gateways
			if test.expectedSize > 2 {
				// create accounts for all the expected delegated gateways
				// and add them to the map
				for i := 0; i < test.expectedSize-1; i++ {
					gatewayAcc := newAccount("secp256k1")
					accMap[gatewayAcc.address] = gatewayAcc.pubKey
				}
			}
			// add the application's account and the accounts of all its
			// delegated gateways to the testing state
			testqueryclients.AddAddressToApplicationMap(t, test.appAccount.address, test.appAccount.pubKey, accMap)
			// Attempt to retrieve the ring for the address and cache it
			ring1, err := rc.GetRingForAddress(ctx, test.appAccount.address)
			require.NoError(t, err)
			require.Equal(t, test.expectedSize, ring1.Size())
			// Attempt to retrieve the ring for the address after its been cached
			ring2, err := rc.GetRingForAddress(ctx, test.appAccount.address)
			require.NoError(t, err)
			// Ensure the rings are the same and have the same size
			require.True(t, ring1.Equals(ring2))
			require.Equal(t, test.expectedSize, ring2.Size())
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
