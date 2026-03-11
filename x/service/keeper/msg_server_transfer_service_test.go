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

func TestMsgServer_TransferService(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	ownerAddr := sample.AccAddressBech32()
	newOwnerAddr := sample.AccAddressBech32()
	nonOwnerAddr := sample.AccAddressBech32()

	// Add balance and create a service owned by ownerAddr.
	keepertest.AddAccToAccMapCoins(t, ownerAddr, pocket.DenomuPOKT, oneUPOKTGreaterThanFee)
	_, err := srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: ownerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-transfer",
			Name:                 "transfer test service",
			ComputeUnitsPerRelay: 1,
			OwnerAddress:         ownerAddr,
		},
	})
	require.NoError(t, err)

	tests := []struct {
		desc            string
		msg             *types.MsgTransferService
		expectedErr     error
		expectedErrCode bool // if true, check by error string contains
	}{
		{
			desc: "invalid - empty owner address",
			msg: types.NewMsgTransferService(
				"", "svc-transfer", newOwnerAddr,
			),
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc: "invalid - empty new owner address",
			msg: types.NewMsgTransferService(
				ownerAddr, "svc-transfer", "",
			),
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc: "invalid - empty service ID",
			msg: types.NewMsgTransferService(
				ownerAddr, "", newOwnerAddr,
			),
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc: "invalid - same owner and new owner",
			msg: types.NewMsgTransferService(
				ownerAddr, "svc-transfer", ownerAddr,
			),
			expectedErr: types.ErrServiceInvalidOwnerAddress,
		},
		{
			desc: "invalid - service does not exist",
			msg: types.NewMsgTransferService(
				ownerAddr, "nonexistent-svc", newOwnerAddr,
			),
			expectedErr: types.ErrServiceNotFound,
		},
		{
			desc: "invalid - non-owner tries to transfer",
			msg: types.NewMsgTransferService(
				nonOwnerAddr, "svc-transfer", newOwnerAddr,
			),
			expectedErr: types.ErrServiceUnauthorized,
		},
		{
			desc: "valid - owner transfers service",
			msg: types.NewMsgTransferService(
				ownerAddr, "svc-transfer", newOwnerAddr,
			),
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, transferErr := srv.TransferService(ctx, test.msg)
			if test.expectedErr != nil {
				require.ErrorAs(t, transferErr, &test.expectedErr)
				return
			}
			require.NoError(t, transferErr)

			// Verify the service now has the new owner.
			svc, found := k.GetService(ctx, test.msg.ServiceId)
			require.True(t, found)
			require.Equal(t, test.msg.NewOwnerAddress, svc.OwnerAddress)
		})
	}

	// After the valid transfer, verify:
	// 1. New owner can update the service.
	_, err = srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: newOwnerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-transfer",
			Name:                 "updated by new owner",
			ComputeUnitsPerRelay: 5,
			OwnerAddress:         newOwnerAddr,
		},
	})
	require.NoError(t, err, "new owner should be able to update the service")

	svc, found := k.GetService(ctx, "svc-transfer")
	require.True(t, found)
	require.Equal(t, "updated by new owner", svc.Name)
	require.Equal(t, uint64(5), svc.ComputeUnitsPerRelay)

	// 2. Old owner can no longer update the service.
	_, err = srv.AddService(ctx, &types.MsgAddService{
		OwnerAddress: ownerAddr,
		Service: sharedtypes.Service{
			Id:                   "svc-transfer",
			Name:                 "should fail",
			ComputeUnitsPerRelay: 10,
			OwnerAddress:         ownerAddr,
		},
	})
	require.Error(t, err, "old owner should not be able to update the service")
	require.ErrorContains(t, err, "invalid owner address")
}
