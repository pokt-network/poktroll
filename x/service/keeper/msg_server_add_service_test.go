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

	// Generate a valid address
	addr := sample.AccAddress()
	// Create a service
	preExistingService := sharedtypes.Service{
		Id:   "svc2",
		Name: "service 2",
	}
	// Add the service to the store
	_, err := srv.AddService(wctx, &types.MsgAddService{
		Address: addr,
		Service: preExistingService,
	})
	require.NoError(t, err)
	// Validate the service was added
	serviceFound, found := k.GetService(ctx, preExistingService.Id)
	require.True(t, found)
	require.Equal(t, preExistingService, serviceFound)

	tests := []struct {
		desc          string
		address       string
		service       sharedtypes.Service
		expectedError error
	}{
		{
			desc:    "valid - service added successfully",
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: nil,
		},
		{
			desc:    "invalid - service supplier address is empty",
			address: "", // explicitly set to empty string
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:    "invalid - invalid service supplier address",
			address: "invalid address",
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:    "invalid - missing service ID",
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				// Explicitly omitting Id field
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - empty service ID",
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "", // explicitly set to empty string
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - missing service name",
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id: "svc1",
				// Explicitly omitting Name field
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:    "invalid - empty service name",
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "", // explicitly set to empty string
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:          "invalid - service already exists (same service supplier)",
			address:       addr,
			service:       preExistingService,
			expectedError: types.ErrServiceAlreadyExists,
		},
		{
			desc:          "invalid - service already exists (different service supplier)",
			address:       sample.AccAddress(),
			service:       preExistingService,
			expectedError: types.ErrServiceAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := srv.AddService(wctx, &types.MsgAddService{
				Address: tt.address,
				Service: tt.service,
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
