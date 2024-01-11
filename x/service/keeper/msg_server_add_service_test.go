package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_AddService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate a supplier address
	supplierAddr := sample.AccAddress()
	// Create a service
	preExistingService := sharedtypes.Service{
		Id:   "svc2",
		Name: "service 2",
	}
	// Add the service to the store
	_, err := srv.AddService(wctx, &types.MsgAddService{
		SupplierAddress: supplierAddr,
		Service:         preExistingService,
	})
	require.NoError(t, err)
	// Validate the service was added
	serviceFound, found := k.GetService(ctx, preExistingService.Id)
	require.True(t, found)
	require.Equal(t, preExistingService, serviceFound)

	tests := []struct {
		desc            string
		supplierAddress string
		service         sharedtypes.Service
		expectedError   error
	}{
		{
			desc:            "valid - service added successfully",
			supplierAddress: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: nil,
		},
		{
			desc:            "invalid - supplier address is empty",
			supplierAddress: "", // explicitly set to empty string
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:            "invalid - invalid supplier address",
			supplierAddress: "invalid address",
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:            "invalid - missing service ID",
			supplierAddress: sample.AccAddress(),
			service: sharedtypes.Service{
				// Explicitly omitting Id field
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:            "invalid - empty service ID",
			supplierAddress: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "", // explicitly set to empty string
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:            "invalid - missing service name",
			supplierAddress: sample.AccAddress(),
			service: sharedtypes.Service{
				Id: "svc1",
				// Explicitly omitting Name field
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:            "invalid - empty service name",
			supplierAddress: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "", // explicitly set to empty string
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:            "invalid - service already exists (same supplier)",
			supplierAddress: supplierAddr,
			service:         preExistingService,
			expectedError:   types.ErrServiceAlreadyExists,
		},
		{
			desc:            "invalid - service already exists (different supplier)",
			supplierAddress: sample.AccAddress(),
			service:         preExistingService,
			expectedError:   types.ErrServiceAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := srv.AddService(wctx, &types.MsgAddService{
				SupplierAddress: tt.supplierAddress,
				Service:         tt.service,
			})
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			// Validate the service was added
			serviceFound, found := k.GetService(ctx, tt.service.Id)
			require.True(t, found)
			require.Equal(t, tt.service, serviceFound)
		})
	}
}
