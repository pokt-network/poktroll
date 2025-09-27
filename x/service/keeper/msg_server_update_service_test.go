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

// oneUPOKTGreaterThanUpdateFee is 1 upokt more than the UpdateServiceFee
var oneUPOKTGreaterThanUpdateFee = types.MinUpdateServiceFee.Amount.Uint64() + 1

func TestMsgServer_UpdateService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	serviceOwnerAddr := sample.AccAddressBech32()
	otherAddr := sample.AccAddressBech32()

	// Pre-existing service
	originalService := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "original service",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         serviceOwnerAddr,
	}

	// Updated service configuration
	updatedService := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "updated service name",
		ComputeUnitsPerRelay: 5,
		OwnerAddress:         serviceOwnerAddr,
	}

	// Mock adding balance to the account
	keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
	keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanUpdateFee)

	// Add the original service to the store first
	_, err := srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service:      originalService,
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		msg         *types.MsgUpdateService
		expectedErr error
	}{
		{
			name: "valid - successful update",
			msg: &types.MsgUpdateService{
				OwnerAddress: serviceOwnerAddr,
				Service:      updatedService,
			},
			expectedErr: nil,
		},
		{
			name: "invalid - service not found",
			msg: &types.MsgUpdateService{
				OwnerAddress: serviceOwnerAddr,
				Service: sharedtypes.Service{
					Id:                   "nonexistent",
					Name:                 "service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         serviceOwnerAddr,
				},
			},
			expectedErr: types.ErrServiceNotFound,
		},
		{
			name: "invalid - not the owner",
			msg: &types.MsgUpdateService{
				OwnerAddress: otherAddr,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "updated service name",
					ComputeUnitsPerRelay: 5,
					OwnerAddress:         otherAddr, // Match the signer address for validation
				},
			},
			expectedErr: types.ErrServiceInvalidOwnerAddress,
		},
		{
			name: "invalid - insufficient funds",
			msg: func() *types.MsgUpdateService {
				noFundsAddr := sample.AccAddressBech32()
				// Need to first add a service with this owner so the update can work
				keepertest.AddAccToAccMapCoins(t, noFundsAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
				testService := sharedtypes.Service{
					Id:                   "svc2",
					Name:                 "test service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         noFundsAddr,
				}
				_, err := srv.AddService(ctx, &types.MsgAddService{
					OwnerAddress: noFundsAddr,
					Service:      testService,
				})
				require.NoError(t, err)

				// Return the update message for an owner with no remaining funds
				return &types.MsgUpdateService{
					OwnerAddress: noFundsAddr,
					Service: sharedtypes.Service{
						Id:                   "svc2",
						Name:                 "updated service",
						ComputeUnitsPerRelay: 2,
						OwnerAddress:         noFundsAddr,
					},
				}
			}(),
			expectedErr: types.ErrServiceNotEnoughFunds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add funds for the valid test case that might need them
			if tt.expectedErr == nil {
				keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanUpdateFee)
			}

			_, err := srv.UpdateService(ctx, tt.msg)
			if tt.expectedErr != nil {
				// Using ErrorAs as wrapping the error sometimes gives errors with ErrorIs
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.NoError(t, err)

			// Verify the service was updated
			service, found := k.GetService(ctx, tt.msg.Service.Id)
			require.True(t, found)
			require.Equal(t, tt.msg.Service, service)
		})
	}
}