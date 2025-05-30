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

var addServiceFee = types.MinAddServiceFee.Amount.Int64()

func TestMsgServer_SetupService(t *testing.T) {
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
	keepertest.AddAccToAccMapCoins(t, oldServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee)

	// Add the service to the store
	addSvcRes, err := srv.SetupService(ctx, &types.MsgSetupService{
		Signer:  oldServiceOwnerAddr,
		Service: oldService,
	})
	require.NoError(t, err)
	require.Equal(t, &oldService, addSvcRes.GetService())

	// Validate the service was added
	serviceFound, found := k.GetService(ctx, oldService.Id)
	require.True(t, found)
	require.Equal(t, oldService, serviceFound)

	tests := []struct {
		desc          string
		setup         func(t *testing.T)
		signerAddress string
		service       sharedtypes.Service
		expectedErr   error
	}{
		{
			desc:          "invalid - service signer address is empty",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: "", // explicitly set to empty string
			service:       newService,
			expectedErr:   types.ErrServiceInvalidAddress,
		},
		{
			desc:          "invalid - invalid service signer address",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: "invalid address",
			service:       newService,
			expectedErr:   types.ErrServiceInvalidAddress,
		},
		{
			desc:          "invalid - missing service ID",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: newServiceOwnerAddr,
			service: sharedtypes.Service{
				// Explicitly omitting Id field
				Name:                 newService.Name,
				ComputeUnitsPerRelay: newService.ComputeUnitsPerRelay,
				OwnerAddress:         newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc:          "invalid - empty service ID",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   "", // explicitly set to empty string
				Name:                 newService.Name,
				ComputeUnitsPerRelay: newService.ComputeUnitsPerRelay,
				OwnerAddress:         newServiceOwnerAddr,
			},
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc:          "invalid - zero compute units per relay",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: newServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   newService.Id,
				Name:                 newService.Name,
				ComputeUnitsPerRelay: 0,
				OwnerAddress:         newServiceOwnerAddr,
			},
			expectedErr: sharedtypes.ErrSharedInvalidComputeUnitsPerRelay,
		},
		{
			desc: "invalid - insufficient upokt balance",
			// Set the balance of the new service owner address to be less than the AddServiceFee param
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee-1),
			signerAddress: newServiceOwnerAddr,
			service:       newService,
			expectedErr:   types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - sufficient balance of different denom",
			// Adds wrong coins to the account
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomMACT, addServiceFee),
			signerAddress: newServiceOwnerAddr,
			service:       newService,
			expectedErr:   types.ErrServiceNotEnoughFunds,
		},
		{
			// Only the current owner of a service can update it.
			desc:          "invalid - existing service owner address does match new signer address",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee-1),
			signerAddress: newServiceOwnerAddr,
			service:       oldService,
			expectedErr:   types.ErrServiceInvalidOwnerAddress,
		},
		{
			desc:          "valid - service added successfully",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: newServiceOwnerAddr,
			service:       newService,
			expectedErr:   nil,
		},
		{
			desc:          "valid - update compute_units_per_relay",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: oldServiceOwnerAddr,
			service: sharedtypes.Service{
				Id:                   oldService.Id,
				Name:                 oldService.Name,
				ComputeUnitsPerRelay: 2, // Update to a new value
				OwnerAddress:         oldServiceOwnerAddr,
			},
		},
		{
			desc:          "valid - update service name",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: oldServiceOwnerAddr,
			service: sharedtypes.Service{
				Id: oldService.Id,
				// Empty service name is allowed
				Name:                 "", // Update to a new empty value
				ComputeUnitsPerRelay: oldService.ComputeUnitsPerRelay,
				OwnerAddress:         oldServiceOwnerAddr,
			},
		},
		{
			desc:          "valid - previous owner changes the service owner address",
			setup:         setupAccountBalance(t, newServiceOwnerAddr, pocket.DenomuPOKT, addServiceFee),
			signerAddress: oldServiceOwnerAddr, // The previous owner is the one updating the service
			service: sharedtypes.Service{
				Id:                   oldService.Id,
				Name:                 oldService.Name,
				ComputeUnitsPerRelay: oldService.ComputeUnitsPerRelay,
				OwnerAddress:         newServiceOwnerAddr, // Change the owner address to the new one
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.setup(t)
			_, err := srv.SetupService(ctx, &types.MsgSetupService{
				Signer:  test.signerAddress,
				Service: test.service,
			})
			if test.expectedErr != nil {
				require.Error(t, err)
				// Check if the error matches the expected error
				// Using ErrorAs as wrapping the error sometimes gives errors with ErrorIs
				require.ErrorAsf(t, err, &test.expectedErr, "expected error %s, got %s", test.expectedErr, err)
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

// setupAccountBalance is a helper factory function that creates a setup function
// to add a specified amount of coins to an account's balance for testing purposes.
func setupAccountBalance(t *testing.T, addr string, denom string, amount int64) func(*testing.T) {
	t.Helper()
	return func(t *testing.T) {
		// Add the specified amount of coins to the account
		keepertest.AddAccToAccMapCoins(t, addr, denom, amount)
	}
}
