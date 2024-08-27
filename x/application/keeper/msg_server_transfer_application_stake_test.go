package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_TransferApplicationStake_Success(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	// Generate an address for the source and destination applications.
	srcAddr := sample.AccAddress()
	dstAddr := sample.AccAddress()

	// Verify that the app does not exist yet.
	_, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare the application.
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: srcAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application.
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists.
	srcApp, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.True(t, isSrcFound)
	require.Equal(t, srcAddr, srcApp.Address)
	require.Equal(t, expectedAppStake, srcApp.Stake)
	require.Len(t, srcApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", srcApp.ServiceConfigs[0].Service.Id)

	// Transfer the application stake from the source to the destination application address.
	transferStakeMsg := apptypes.NewMsgTransferApplicationStake(srcAddr, dstAddr)

	transferAppStakeRes, stakeTransferErr := srv.TransferApplicationStake(ctx, transferStakeMsg)
	require.NoError(t, stakeTransferErr)

	// Verify that the destination app was created with the correct state.
	srcApp, isSrcFound = k.GetApplication(ctx, dstAddr)
	require.True(t, isSrcFound)

	dstApp, isDstFound := k.GetApplication(ctx, dstAddr)
	require.True(t, isDstFound)
	require.Equal(t, dstAddr, dstApp.Address)
	require.Equal(t, expectedAppStake, dstApp.Stake)
	require.Len(t, dstApp.ServiceConfigs, 1)
	require.EqualValues(t, srcApp, dstApp)
	require.EqualValues(t, &dstApp, transferAppStakeRes.Application)

	// Verify that the source app was unstaked.
	srcApp, isSrcFound = k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)
}

func TestMsgServer_TransferApplicationStake_Error_DestinationExists(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	// Generate an address for the source and destination applications.
	srcAddr := sample.AccAddress()
	dstAddr := sample.AccAddress()

	// Verify that neither the source nor the destination application exists yet.
	_, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)

	_, isDstFound := k.GetApplication(ctx, dstAddr)
	require.False(t, isDstFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare and stake the application.
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address: srcAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	_, err := srv.StakeApplication(ctx, appStakeMsg)
	require.NoError(t, err)

	// Prepare and stake the destination application.
	dstAppStakeMsg := &apptypes.MsgStakeApplication{
		Address: dstAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	_, err = srv.StakeApplication(ctx, dstAppStakeMsg)
	require.NoError(t, err)

	// Attempt to transfer the source application stake to the destination.
	transferStakeMsg := apptypes.NewMsgTransferApplicationStake(srcAddr, dstAddr)

	_, err = srv.TransferApplicationStake(ctx, transferStakeMsg)
	require.ErrorContains(t, err, apptypes.ErrAppDuplicateAddress.Wrapf("destination application (%q) exists", dstAddr).Error())

	// Verify that the original application still exists.
	var foundApp apptypes.Application
	foundApp, isSrcFound = k.GetApplication(ctx, srcAddr)
	require.True(t, isSrcFound)
	require.Equal(t, srcAddr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)
}
