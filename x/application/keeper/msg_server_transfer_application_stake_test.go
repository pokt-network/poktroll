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

	// Generate an address for the application and beneficiary.
	appAddr := sample.AccAddress()
	beneficiaryAddr := sample.AccAddress()

	// Verify that the app does not exist yet.
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare the application.
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
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
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, expectedAppStake, foundApp.Stake)
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)

	// Transfer the application stake to the beneficiary.
	transferStakeMsg := apptypes.NewMsgTransferApplicationStake(appAddr, beneficiaryAddr)

	_, err = srv.TransferApplicationStake(ctx, transferStakeMsg)
	require.NoError(t, err)

	// Verify that the beneficiary was created with the same stake and service configs.
	foundApp, isAppFound = k.GetApplication(ctx, beneficiaryAddr)
	foundBeneficiary, isBeneficiaryFound := k.GetApplication(ctx, beneficiaryAddr)
	require.True(t, isBeneficiaryFound)
	require.Equal(t, beneficiaryAddr, foundBeneficiary.Address)
	require.Equal(t, expectedAppStake, foundBeneficiary.Stake)
	require.Len(t, foundBeneficiary.ServiceConfigs, 1)
	require.EqualValues(t, foundApp.ServiceConfigs[0], foundBeneficiary.ServiceConfigs[0])

	// Verify that the original app was unstaked.
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)
}

func TestMsgServer_TransferApplicationStake_Error_BeneficiaryExists(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	// Generate an address for the application and beneficiary.
	appAddr := sample.AccAddress()
	beneficiaryAddr := sample.AccAddress()

	// Verify that neither the app nor the beneficiary exists yet.
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)

	_, isBeneficiaryFound := k.GetApplication(ctx, beneficiaryAddr)
	require.False(t, isBeneficiaryFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare and stake the application.
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	_, err := srv.StakeApplication(ctx, appStakeMsg)
	require.NoError(t, err)

	// Prepare and stake the beneficiary.
	beneficiaryStakeMsg := &apptypes.MsgStakeApplication{
		Address: beneficiaryAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	_, err = srv.StakeApplication(ctx, beneficiaryStakeMsg)
	require.NoError(t, err)

	// Attempt to transfer the application stake to the beneficiary.
	transferStakeMsg := apptypes.NewMsgTransferApplicationStake(appAddr, beneficiaryAddr)

	_, err = srv.TransferApplicationStake(ctx, transferStakeMsg)
	require.ErrorContains(t, err, apptypes.ErrAppDuplicateAddress.Wrapf("beneficiary (%q) exists", beneficiaryAddr).Error())

	// Verify that the original application still exists.
	var foundApp apptypes.Application
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)
}
