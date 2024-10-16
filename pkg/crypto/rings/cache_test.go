package rings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	testrings "github.com/pokt-network/poktroll/testutil/testcrypto/rings"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

const (
	noDelegateesRingSize = 2
	defaultHeight        = 1
)

// account is an internal struct used to define an (address, public_key) pairing
type account struct {
	address string
	pubKey  cryptotypes.PubKey
}

// newAccount creates a new account for testing purposes on the desired curve
func newAccount(curve string) account {
	var addr string
	var pubkey cryptotypes.PubKey
	switch curve {
	case "ed25519":
		addr, pubkey = sample.AccAddressAndPubKeyEd25519()
	case "secp256k1":
		addr, pubkey = sample.AccAddressAndPubKey()
	}
	return account{
		address: addr,
		pubKey:  pubkey,
	}
}

func TestRingCache_BuildRing_Uncached(t *testing.T) {
	// Create and start the ring cache
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	rc, _ := createRingCache(ctx, t, "")
	rc.Start(ctx)
	t.Cleanup(rc.Stop)

	tests := []struct {
		desc              string
		appAddrIndex      int
		appAccount        account
		delegateeAccounts []account
		expectedRingSize  int
		expectedErr       error
	}{
		{
			desc:              "success: un-cached application without delegated gateways",
			appAddrIndex:      1,
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{},
			expectedRingSize:  noDelegateesRingSize,
			expectedErr:       nil,
		},
		{
			desc:              "success: un-cached application with delegated gateways",
			appAddrIndex:      2,
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("secp256k1"), newAccount("secp256k1")},
			expectedRingSize:  3,
			expectedErr:       nil,
		},
		{
			desc:              "failure: app pubkey uses wrong curve",
			appAccount:        newAccount("ed25519"),
			delegateeAccounts: []account{newAccount("secp256k1"), newAccount("secp256k1")},
			expectedRingSize:  0,
			expectedErr:       rings.ErrRingsNotSecp256k1Curve,
		},
		{
			desc:              "failure: gateway pubkey uses wrong curve",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("ed25519"), newAccount("ed25519")},
			expectedRingSize:  0,
			expectedErr:       rings.ErrRingsNotSecp256k1Curve,
		},
		{
			desc:              "failure: application not found",
			appAccount:        newAccount("secp256k1"),
			delegateeAccounts: []account{newAccount("secp256k1")},
			expectedRingSize:  0,
			expectedErr:       apptypes.ErrAppNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// If we expect the application to exist then add it to the test
			// application map with the number of delegated gateways it is
			// supposed to have so it can be retrieved from the mock
			if !errors.As(test.expectedErr, &apptypes.ErrAppNotFound) {
				accMap := make(map[string]cryptotypes.PubKey)
				for _, delegateeAcc := range test.delegateeAccounts {
					accMap[delegateeAcc.address] = delegateeAcc.pubKey
				}
				// add the application's account and the accounts of all its
				// delegated gateways to the testing state
				testqueryclients.AddAddressToApplicationMap(t, test.appAccount.address, test.appAccount.pubKey, accMap)
			}
			// Attempt to retrieve the ring for the address
			ring, err := rc.GetRingForAddressAtHeight(ctx, test.appAccount.address, defaultHeight)
			if test.expectedErr != nil {
				require.ErrorAs(t, err, &test.expectedErr)
				return
			}
			require.NoError(t, err)
			// Ensure the ring is the correct size.
			require.Equal(t, test.expectedRingSize, ring.Size())
			require.Equal(t, test.appAddrIndex, len(rc.GetCachedAddresses()))
		})
	}
}

func TestRingCache_BuildRing_Cached(t *testing.T) {
	tests := []struct {
		desc             string
		appAccount       account
		expectedRingSize int
		expectedErr      error
	}{
		{
			desc:             "success: cached application without delegated gateways",
			appAccount:       newAccount("secp256k1"),
			expectedRingSize: noDelegateesRingSize,
			expectedErr:      nil,
		},
		{
			desc:             "success: cached application with delegated gateways",
			appAccount:       newAccount("secp256k1"),
			expectedRingSize: 3,
			expectedErr:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Create and start the ring cache
			ctx, cancelCtx := context.WithCancel(context.Background())
			defer cancelCtx()
			rc, pubCh := createRingCache(ctx, t, test.appAccount.address)
			rc.Start(ctx)
			t.Cleanup(rc.Stop)

			// Check that the ring cache is empty
			require.Equal(t, 0, len(rc.GetCachedAddresses()))

			// add the application's account with no delegated gateways to the
			// testing state
			testqueryclients.AddAddressToApplicationMap(t, test.appAccount.address, test.appAccount.pubKey, nil)

			// Attempt to retrieve the ring for the address and cache it
			ring1, err := rc.GetRingForAddressAtHeight(ctx, test.appAccount.address, defaultHeight)
			require.NoError(t, err)
			require.Equal(t, noDelegateesRingSize, ring1.Size())
			require.Equal(t, 1, len(rc.GetCachedAddresses()))

			gatewayAddrToPubKeyMap := make(map[string]cryptotypes.PubKey)
			// if the test expects a ring > 2 we have delegated gateways
			if test.expectedRingSize != noDelegateesRingSize {
				// create accounts for all the expected delegated gateways
				// and add them to the map
				for i := 0; i < test.expectedRingSize-1; i++ {
					gatewayAcc := newAccount("secp256k1")
					gatewayAddrToPubKeyMap[gatewayAcc.address] = gatewayAcc.pubKey
				}
			}

			// add the application's account and the accounts of all its
			// delegated gateways to the testing state simulating a change
			testqueryclients.AddAddressToApplicationMap(t, test.appAccount.address, test.appAccount.pubKey, gatewayAddrToPubKeyMap)
			for gatewayAddr := range gatewayAddrToPubKeyMap {
				t.Log(gatewayAddrToPubKeyMap)
				// publish a redelegation event
				pubCh <- &apptypes.EventRedelegation{
					Application: &apptypes.Application{
						Address:                   test.appAccount.address,
						DelegateeGatewayAddresses: []string{gatewayAddr},
					},
				}
			}

			// Wait a tick to allow the ring cache to process asynchronously.
			// It should have invalidated the cache for the ring, if changed.
			time.Sleep(15 * time.Millisecond)

			// Attempt to retrieve the ring for the address and cache it if
			// the ring was updated
			ring2, err := rc.GetRingForAddressAtHeight(ctx, test.appAccount.address, defaultHeight)
			require.NoError(t, err)
			// If the ring was updated then the rings should not be equal
			if test.expectedRingSize != noDelegateesRingSize {
				require.False(t, ring1.Equals(ring2))
			} else {
				require.True(t, ring1.Equals(ring2))
			}
			require.Equal(t, test.expectedRingSize, ring2.Size())
			require.Equal(t, 1, len(rc.GetCachedAddresses()))

			// Attempt to retrieve the ring for the address after its been cached
			ring3, err := rc.GetRingForAddressAtHeight(ctx, test.appAccount.address, defaultHeight)
			require.NoError(t, err)
			require.Equal(t, 1, len(rc.GetCachedAddresses()))

			// Ensure the rings are the same and have the same size
			require.True(t, ring2.Equals(ring3))
			require.Equal(t, test.expectedRingSize, ring3.Size())
			require.Equal(t, 1, len(rc.GetCachedAddresses()))
		})
	}
}

func TestRingCache_Stop(t *testing.T) {
	// Create and start the ring cache
	ctx, cancelCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelCtx)
	rc, _ := createRingCache(ctx, t, "")
	rc.Start(ctx)

	// Insert an application into the testing state
	appAccount := newAccount("secp256k1")
	gatewayAccount := newAccount("secp256k1")
	testqueryclients.AddAddressToApplicationMap(
		t, appAccount.address,
		appAccount.pubKey,
		map[string]cryptotypes.PubKey{
			gatewayAccount.address: gatewayAccount.pubKey,
		})

	// Attempt to retrieve the ring for the address and cache it
	ring1, err := rc.GetRingForAddressAtHeight(ctx, appAccount.address, defaultHeight)
	require.NoError(t, err)
	require.Equal(t, 2, ring1.Size())
	require.Equal(t, 1, len(rc.GetCachedAddresses()))

	// Retrieve the cached ring
	ring2, err := rc.GetRingForAddressAtHeight(ctx, appAccount.address, defaultHeight)
	require.NoError(t, err)
	require.True(t, ring1.Equals(ring2))
	require.Equal(t, 1, len(rc.GetCachedAddresses()))

	// Stop the ring cache
	rc.Stop()

	// Retrieve the ring again
	require.Equal(t, 0, len(rc.GetCachedAddresses()))
}

func TestRingCache_CancelContext(t *testing.T) {
	// Create and start the ring cache
	ctx, cancelCtx := context.WithCancel(context.Background())
	rc, _ := createRingCache(ctx, t, "")
	rc.Start(ctx)

	// Insert an application into the testing state
	appAccount := newAccount("secp256k1")
	gatewayAccount := newAccount("secp256k1")
	testqueryclients.AddAddressToApplicationMap(
		t,
		appAccount.address, appAccount.pubKey,
		map[string]cryptotypes.PubKey{
			gatewayAccount.address: gatewayAccount.pubKey,
		})

	// Attempt to retrieve the ring for the address and cache it
	ring1, err := rc.GetRingForAddressAtHeight(ctx, appAccount.address, defaultHeight)
	require.NoError(t, err)
	require.Equal(t, 2, ring1.Size())
	require.Equal(t, 1, len(rc.GetCachedAddresses()))

	// Retrieve the cached ring
	ring2, err := rc.GetRingForAddressAtHeight(ctx, appAccount.address, defaultHeight)
	require.NoError(t, err)
	require.True(t, ring1.Equals(ring2))
	require.Equal(t, 1, len(rc.GetCachedAddresses()))

	// Cancel the context
	cancelCtx()

	// Wait a tick to allow the ring cache to process asynchronously.
	time.Sleep(15 * time.Millisecond)

	// Retrieve the ring again
	require.Equal(t, 0, len(rc.GetCachedAddresses()))
}

// createRingCache creates the RingCache using mocked AccountQueryClient and
// ApplicatioQueryClient instances and returns the RingCache and the delegatee
// change replay observable.
func createRingCache(ctx context.Context, t *testing.T, appAddress string) (crypto.RingCache, chan<- *apptypes.EventRedelegation) {
	t.Helper()
	redelegationObs, redelegationPublishCh := channel.NewReplayObservable[*apptypes.EventRedelegation](ctx, 1)
	delegationClient := testdelegation.NewAnyTimesRedelegationsSequence(ctx, t, appAddress, redelegationObs)
	accQuerier := testqueryclients.NewTestAccountQueryClient(t)
	appQuerier := testqueryclients.NewTestApplicationQueryClient(t)
	sharedQuerier := testqueryclients.NewTestSharedQueryClient(t)

	ringCacheDeps := depinject.Supply(accQuerier, appQuerier, delegationClient, sharedQuerier)
	rc := testrings.NewRingCacheWithMockDependencies(ctx, t, ringCacheDeps)
	return rc, redelegationPublishCh
}
