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

func TestMsgServer_DeleteService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	serviceOwnerAddr := sample.AccAddressBech32()
	otherAddr := sample.AccAddressBech32()

	// Pre-existing service
	existingService := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "service to delete",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         serviceOwnerAddr,
	}

	// Mock adding balance to the account for adding the service first
	keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)

	// Add the service to the store first
	_, err := srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service:      existingService,
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		msg         *types.MsgDeleteService
		expectedErr error
	}{
		{
			name: "valid - successful delete",
			msg: &types.MsgDeleteService{
				OwnerAddress: serviceOwnerAddr,
				ServiceId:    "svc1",
			},
			expectedErr: nil,
		},
		{
			name: "invalid - service not found",
			msg: &types.MsgDeleteService{
				OwnerAddress: serviceOwnerAddr,
				ServiceId:    "nonexistent",
			},
			expectedErr: types.ErrServiceNotFound,
		},
		{
			name: "invalid - not the owner",
			msg: &types.MsgDeleteService{
				OwnerAddress: otherAddr,
				ServiceId:    "svc1",
			},
			expectedErr: types.ErrServiceInvalidOwnerAddress,
		},
		{
			name: "invalid - invalid owner address",
			msg: &types.MsgDeleteService{
				OwnerAddress: "invalid_address",
				ServiceId:    "svc1",
			},
			expectedErr: types.ErrServiceInvalidAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For tests that expect the service to exist, re-add it
			if tt.name == "valid - successful delete" || tt.name == "invalid - not the owner" || tt.name == "invalid - invalid owner address" {
				// Re-add service for tests that need it to exist
				keepertest.AddAccToAccMapCoins(t, serviceOwnerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
				_, err := srv.AddService(ctx, &types.MsgAddService{
					OwnerAddress: serviceOwnerAddr,
					Service:      existingService,
				})
				require.NoError(t, err)
			}

			_, err := srv.DeleteService(ctx, tt.msg)
			if tt.expectedErr != nil {
				// Using ErrorAs as wrapping the error sometimes gives errors with ErrorIs
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.NoError(t, err)

			// Verify the service was deleted
			_, found := k.GetService(ctx, tt.msg.ServiceId)
			require.False(t, found)
		})
	}
}