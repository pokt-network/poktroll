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

// oneUPOKTGreaterThanFee is 1 upokt more than the AddServiceFee of 10000000000
const oneUPOKTGreaterThanFee = 10000000001

func TestMsgServer_AddService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Create a service
	srv1 := sharedtypes.Service{
		Id:   "svc1",
		Name: "service 1",
	}

	// Generate a valid address
	addr := sample.AccAddress()
	// Create a service
	preExistingService := sharedtypes.Service{
		Id:   "svc2",
		Name: "service 2",
	}
	// Mock adding a balance to the account
	keepertest.AddAccToAccMapCoins(t, addr, "upokt", oneUPOKTGreaterThanFee)
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

	validAddr1 := sample.AccAddress()
	validAddr2 := sample.AccAddress()

	tests := []struct {
		desc          string
		preFn         func(t *testing.T)
		address       string
		service       sharedtypes.Service
		expectedError error
	}{
		{
			desc: "valid - service added successfully",
			preFn: func(t *testing.T) {
				// Add 10000000001 upokt to the account
				keepertest.AddAccToAccMapCoins(t, validAddr1, "upokt", oneUPOKTGreaterThanFee)
			},
			address:       validAddr1,
			service:       srv1,
			expectedError: nil,
		},
		{
			desc:    "invalid - service supplier address is empty",
			preFn:   func(t *testing.T) {},
			address: "", // explicitly set to empty string
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:          "invalid - invalid service supplier address",
			preFn:         func(t *testing.T) {},
			address:       "invalid address",
			service:       srv1,
			expectedError: types.ErrServiceInvalidAddress,
		},
		{
			desc:    "invalid - missing service ID",
			preFn:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				// Explicitly omitting Id field
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - empty service ID",
			preFn:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "", // explicitly set to empty string
				Name: "service 1",
			},
			expectedError: types.ErrServiceMissingID,
		},
		{
			desc:    "invalid - missing service name",
			preFn:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id: "svc1",
				// Explicitly omitting Name field
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:    "invalid - empty service name",
			preFn:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: sharedtypes.Service{
				Id:   "svc1",
				Name: "", // explicitly set to empty string
			},
			expectedError: types.ErrServiceMissingName,
		},
		{
			desc:          "invalid - service already exists (same service supplier)",
			preFn:         func(t *testing.T) {},
			address:       addr,
			service:       preExistingService,
			expectedError: types.ErrServiceAlreadyExists,
		},
		{
			desc:          "invalid - service already exists (different service supplier)",
			preFn:         func(t *testing.T) {},
			address:       sample.AccAddress(),
			service:       preExistingService,
			expectedError: types.ErrServiceAlreadyExists,
		},
		{
			desc:          "invalid - no spendable coins",
			preFn:         func(t *testing.T) {},
			address:       sample.AccAddress(),
			service:       srv1,
			expectedError: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - insufficient upokt balance",
			preFn: func(t *testing.T) {
				// Add 999999999 upokt to the account (one less than AddServiceFee)
				keepertest.AddAccToAccMapCoins(t, validAddr2, "upokt", oneUPOKTGreaterThanFee-2)
			},
			address:       validAddr2,
			service:       srv1,
			expectedError: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - account has exactly AddServiceFee",
			preFn: func(t *testing.T) {
				// Add 10000000000 upokt to the account
				keepertest.AddAccToAccMapCoins(t, validAddr2, "upokt", oneUPOKTGreaterThanFee-1)
			},
			address:       validAddr2,
			service:       srv1,
			expectedError: types.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - sufficient balance of different denom",
			preFn: func(t *testing.T) {
				// Adds 10000000001 wrong coins to the account
				keepertest.AddAccToAccMapCoins(t, validAddr2, "wrong", oneUPOKTGreaterThanFee)
			},
			address:       validAddr2,
			service:       srv1,
			expectedError: types.ErrServiceNotEnoughFunds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.preFn(t)
			_, err := srv.AddService(wctx, &types.MsgAddService{
				Address: tt.address,
				Service: tt.service,
			})
			if tt.expectedError != nil {
				// Using ErrorAs as wrapping the error sometimes gives errors with ErrorIs
				require.ErrorAs(t, err, &tt.expectedError)
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
