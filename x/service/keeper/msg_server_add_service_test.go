package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// oneUPOKTGreaterThanFee is 1 upokt more than the AddServiceFee
const oneUPOKTGreaterThanFee = types.DefaultAddServiceFee + 1

func TestMsgServer_AddService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	oldServiceOwnerAddr := sample.AccAddress()
	newServiceOwnerAddr := sample.AccAddress()

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
	keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, "upokt", oneUPOKTGreaterThanFee)

	// Add the service to the store
	_, err := srv.AddService(ctx, &types.MsgAddService{
		Address: oldServiceOwnerAddr,
		Service: oldService,
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
			desc: "valid - service added successfully",
			setup: func(t *testing.T) {
				// Add 10000000001 upokt to the account
				keepertest.AddAccToAccMapCoins(t, newServiceOwnerAddr, "upokt", oneUPOKTGreaterThanFee)
			},
			address:     newServiceOwnerAddr,
			service:     newService,
			expectedErr: nil,
		},
		{
			desc:    "invalid - service supplier address is empty",
			setup:   func(t *testing.T) {},
			address: "", // explicitly set to empty string
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc:        "invalid - invalid service supplier address",
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
			expectedErr: types.ErrServiceInvalidComputeUnitsPerRelay,
		},
		{
			desc:        "invalid - service already exists (same service supplier)",
			setup:       func(t *testing.T) {},
			address:     newServiceOwnerAddr,
			service:     oldService,
			expectedErr: types.ErrServiceAlreadyExists,
		},
		{
			desc:        "invalid - service already exists (different service supplier)",
			setup:       func(t *testing.T) {},
			address:     newServiceOwnerAddr,
			service:     oldService,
			expectedErr: types.ErrServiceAlreadyExists,
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
				keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, "upokt", oneUPOKTGreaterThanFee-2)
			},
			address:     oldServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - account has exactly AddServiceFee",
			setup: func(t *testing.T) {
				// Add the exact fee in upokt to the account
				keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, "upokt", types.DefaultAddServiceFee)
			},
			address:     oldServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - sufficient balance of different denom",
			setup: func(t *testing.T) {
				// Adds 10000000001 wrong coins to the account
				keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, "wrong", oneUPOKTGreaterThanFee)
			},
			address:     oldServiceOwnerAddr,
			service:     newService,
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.setup(t)
			_, err := srv.AddService(ctx, &types.MsgAddService{
				Address: test.address,
				Service: test.service,
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
