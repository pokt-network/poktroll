package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// oneUPOKTGreaterThanFee is 1 upokt more than the AddServiceFee
var oneUPOKTGreaterThanFee = types.MinAddServiceFee.Amount.Uint64() + 1

func TestMsgServer_AddService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	oldServiceOwnerAddr := sample.AccAddressBech32()
	newServiceOwnerAddr := sample.AccAddressBech32()

	// Pre-existing service
	oldService := sharedtypes.Service{
		Id:                   "svc0",
		Name:                 "service 0",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         oldServiceOwnerAddr,
	}

	// Declare new test service to be added
	newService := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "service 1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         newServiceOwnerAddr,
	}

	// Mock adding a balance to the account
	keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)

	// Add the service to the store
	_, err := srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: oldServiceOwnerAddr,
		Service:      oldService,
	})
	require.NoError(t, err)

	// Validate the service was added
	serviceFound, found := k.GetService(ctx, oldService.Id)
	require.True(t, found)
	require.Equal(t, oldService, serviceFound)

	tests := []struct {
		desc        string
		setup       func(t *testing.T)
		address     string
		service     sharedtypes.Service
		expectedErr error
	}{
		{
			desc:    "invalid - service owner address is empty",
			setup:   func(t *testing.T) {},
			address: "", // explicitly set to empty string
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc:        "invalid - invalid service owner address",
			setup:       func(t *testing.T) {},
			address:     "invalid address",
			service:     newService,
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc:    "invalid - missing service ID",
			setup:   func(t *testing.T) {},
			address: newServiceOwnerAddr,
			service: sharedtypes.Service{
				// Explicitly omitting Id field
				Name:         "service 1",
				OwnerAddress: newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - empty service ID",
			setup:   func(t *testing.T) {},
			address: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:           "", // explicitly set to empty string
				Name:         "service 1",
				OwnerAddress: newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - missing service name",
			setup:   func(t *testing.T) {},
			address: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id: "svc1",
				// Explicitly omitting Name field
				OwnerAddress: newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingName,
		},
		{
			desc:    "invalid - empty service name",
			setup:   func(t *testing.T) {},
			address: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:           "svc1",
				Name:         "", // explicitly set to empty string
				OwnerAddress: newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingName,
		},
		{
			desc:    "invalid - zero compute units per relay",
			setup:   func(t *testing.T) {},
			address: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   "svc1",
				Name:                 "service 1",
				ComputeUnitsPerRelay: 0,
			},
			expectedErr: sharedtypes.ErrSharedInvalidComputeUnitsPerRelay,
		},
		{
			desc:        "invalid - no spendable coins",
			setup:       func(t *testing.T) {},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - insufficient upokt balance",
			setup: func(t *testing.T) {
				// Add 999999999 upokt to the account (one less than AddServiceFee)
				keepertest.AddAccToAccMapCoins(t, newServiceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee-2)
			},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - account has exactly AddServiceFee",
			setup: func(t *testing.T) {
				// Add the exact fee in upokt to the account
				keepertest.AddAccToAccMapCoins(t, newServiceOwnerAddr, pocket.DenomuPOKT, types.MinAddServiceFee.Amount.Uint64())
			},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - sufficient balance of different denom",
			setup: func(t *testing.T) {
				// Adds 10000000001 wrong coins to the account
				keepertest.AddAccToAccMapCoins(t, newServiceOwnerAddr, pocket.DenomMACT, oneUPOKTGreaterThanFee)
			},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc:        "invalid - existing service owner address does match new service address",
			setup:       func(t *testing.T) {},
			address:     newServiceOwnerAddr,
			service:     oldService,
			expectedErr: types.ErrServiceInvalidOwnerAddress,
		},
		// This test is placed after those that check for errors, because of the stateful
		// nature of the test setup.
		// If placed first, those above will follow the update service logic and
		// will not return an error.
		{
			desc: "valid - service added successfully",
			setup: func(t *testing.T) {
				// Add 10000000001 upokt to the account
				keepertest.AddAccToAccMapCoins(t, newServiceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
			},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: nil,
		},
		{
			desc: "valid - update compute_units_per_relay if the owner is correct",
			setup: func(t *testing.T) {
				// Add 10000000001 upokt to the account
				keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
			},
			address: oldServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   oldService.Id,
				Name:                 oldService.Name,
				ComputeUnitsPerRelay: 20, // Update to a new value
				OwnerAddress:         oldServiceOwnerAddr,
			},
			expectedErr: nil,
		},
		{
			desc:    "valid - update service name (human readable description)",
			setup:   func(t *testing.T) {},
			address: oldServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   oldService.Id,
				Name:                 "updated service name",
				ComputeUnitsPerRelay: 20,
				OwnerAddress:         oldServiceOwnerAddr,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.setup(t)
			_, err := srv.AddService(ctx, &types.MsgAddService{
				OwnerAddress: test.address,
				Service:      test.service,
			})
			if test.expectedErr != nil {
				// Using ErrorAs as wrapping the error sometimes gives errors with ErrorIs
				require.ErrorAs(t, err, &test.expectedErr)
				return
			}
			require.NoError(t, err)
			// Validate the service was added
			serviceFound, found := k.GetService(ctx, test.service.Id)
			require.True(t, found)
			require.Equal(t, test.service, serviceFound)
		})
	}
}

// TestMsgServer_AddService_UpdatePreservesMetadata asserts that an update which does
// not carry metadata leaves the stored metadata intact.
//
// MsgAddService is the only update path for an existing service and always carries a
// full Service{}, so a message that only intends to change compute_units_per_relay
// arrives with a nil Metadata. Treating that as "clear the metadata" silently destroys
// onchain state for any client that does not re-send it.
func TestMsgServer_AddService_UpdatePreservesMetadata(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	serviceOwnerAddr := sample.AccAddressBech32()
	keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)

	originalMetadata := &sharedtypes.Metadata{
		Card: []byte(`{"openrpc":"1.2.6"}`),
	}

	// Create the service with metadata.
	_, err := srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-meta",
			Name:                 "service with metadata",
			ComputeUnitsPerRelay: 1,
			OwnerAddress:         serviceOwnerAddr,
			Metadata:             originalMetadata,
		},
	})
	require.NoError(t, err)

	serviceFound, found := k.GetService(ctx, "svc-meta")
	require.True(t, found)
	require.Equal(t, originalMetadata, serviceFound.Metadata)

	// Update ONLY compute_units_per_relay, omitting metadata (what the `edit-service`
	// CLI and any client using NewMsgAddService submits).
	_, err = srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-meta",
			Name:                 "service with metadata",
			ComputeUnitsPerRelay: 42,
			OwnerAddress:         serviceOwnerAddr,
			// Metadata intentionally omitted.
		},
	})
	require.NoError(t, err)

	serviceFound, found = k.GetService(ctx, "svc-meta")
	require.True(t, found)
	require.Equal(t, uint64(42), serviceFound.ComputeUnitsPerRelay, "cupr update must still apply")
	require.Equal(t, originalMetadata, serviceFound.Metadata, "metadata must survive a metadata-less update")

	// An update that DOES carry metadata must still replace it.
	replacementMetadata := &sharedtypes.Metadata{
		Card: []byte(`{"openapi":"3.1.0"}`),
	}
	_, err = srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-meta",
			Name:                 "service with metadata",
			ComputeUnitsPerRelay: 42,
			OwnerAddress:         serviceOwnerAddr,
			Metadata:             replacementMetadata,
		},
	})
	require.NoError(t, err)

	serviceFound, found = k.GetService(ctx, "svc-meta")
	require.True(t, found)
	require.Equal(t, replacementMetadata, serviceFound.Metadata, "explicit metadata must replace the stored value")
}
