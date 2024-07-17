package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
)

// oneUPOKTGreaterThanFee is 1 upokt more than the AddServiceFee
const oneUPOKTGreaterThanFee = service.DefaultAddServiceFee + 1

func TestMsgServer_AddService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Declare test services
	svc1 := shared.Service{
		Id:   "svc1",
		Name: "service 1",
	}

	preExistingService := shared.Service{
		Id:   "svc2",
		Name: "service 2",
	}

	// Generate a valid address
	addr := sample.AccAddress()

	// Mock adding a balance to the account
	keepertest.AddAccToAccMapCoins(t, addr, "upokt", oneUPOKTGreaterThanFee)

	// Add the service to the store
	_, err := srv.AddService(ctx, &service.MsgAddService{
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
		desc        string
		setup       func(t *testing.T)
		address     string
		service     shared.Service
		expectedErr error
	}{
		{
			desc: "valid - service added successfully",
			setup: func(t *testing.T) {
				// Add 10000000001 upokt to the account
				keepertest.AddAccToAccMapCoins(t, validAddr1, "upokt", oneUPOKTGreaterThanFee)
			},
			address:     validAddr1,
			service:     svc1,
			expectedErr: nil,
		},
		{
			desc:    "invalid - service supplier address is empty",
			setup:   func(t *testing.T) {},
			address: "", // explicitly set to empty string
			service: shared.Service{
				Id:   "svc1",
				Name: "service 1",
			},
			expectedErr: service.ErrServiceInvalidAddress,
		},
		{
			desc:        "invalid - invalid service supplier address",
			setup:       func(t *testing.T) {},
			address:     "invalid address",
			service:     svc1,
			expectedErr: service.ErrServiceInvalidAddress,
		},
		{
			desc:    "invalid - missing service ID",
			setup:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: shared.Service{
				// Explicitly omitting Id field
				Name: "service 1",
			},
			expectedErr: service.ErrServiceMissingID,
		},
		{
			desc:    "invalid - empty service ID",
			setup:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: shared.Service{
				Id:   "", // explicitly set to empty string
				Name: "service 1",
			},
			expectedErr: service.ErrServiceMissingID,
		},
		{
			desc:    "invalid - missing service name",
			setup:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: shared.Service{
				Id: "svc1",
				// Explicitly omitting Name field
			},
			expectedErr: service.ErrServiceMissingName,
		},
		{
			desc:    "invalid - empty service name",
			setup:   func(t *testing.T) {},
			address: sample.AccAddress(),
			service: shared.Service{
				Id:   "svc1",
				Name: "", // explicitly set to empty string
			},
			expectedErr: service.ErrServiceMissingName,
		},
		{
			desc:        "invalid - service already exists (same service supplier)",
			setup:       func(t *testing.T) {},
			address:     addr,
			service:     preExistingService,
			expectedErr: service.ErrServiceAlreadyExists,
		},
		{
			desc:        "invalid - service already exists (different service supplier)",
			setup:       func(t *testing.T) {},
			address:     sample.AccAddress(),
			service:     preExistingService,
			expectedErr: service.ErrServiceAlreadyExists,
		},
		{
			desc:        "invalid - no spendable coins",
			setup:       func(t *testing.T) {},
			address:     sample.AccAddress(),
			service:     svc1,
			expectedErr: service.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - insufficient upokt balance",
			setup: func(t *testing.T) {
				// Add 999999999 upokt to the account (one less than AddServiceFee)
				keepertest.AddAccToAccMapCoins(t, validAddr2, "upokt", oneUPOKTGreaterThanFee-2)
			},
			address:     validAddr2,
			service:     svc1,
			expectedErr: service.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - account has exactly AddServiceFee",
			setup: func(t *testing.T) {
				// Add the exact fee in upokt to the account
				keepertest.AddAccToAccMapCoins(t, validAddr2, "upokt", service.DefaultAddServiceFee)
			},
			address:     validAddr2,
			service:     svc1,
			expectedErr: service.ErrServiceNotEnoughFunds,
		},
		{
			desc: "invalid - sufficient balance of different denom",
			setup: func(t *testing.T) {
				// Adds 10000000001 wrong coins to the account
				keepertest.AddAccToAccMapCoins(t, validAddr2, "wrong", oneUPOKTGreaterThanFee)
			},
			address:     validAddr2,
			service:     svc1,
			expectedErr: service.ErrServiceNotEnoughFunds,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.setup(t)
			_, err := srv.AddService(ctx, &service.MsgAddService{
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
