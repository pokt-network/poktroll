package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/migration/mocks"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMsgServer_ClaimMorseApplication_SuccessNewApplication(t *testing.T) {
	shannonDestAddr := sample.AccAddress()
	shannonDestAccAddr, err := sdk.AccAddressFromBech32(shannonDestAddr)
	require.NoError(t, err)

	expectedClaimedAtHeight := int64(10)
	testServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}
	unstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	applicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)
	expectedApp := apptypes.Application{
		Address:        shannonDestAddr,
		Stake:          &applicationStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{testServiceConfig},
	}

	ctrl := gomock.NewController(t)
	bankKeeper := mocks.NewMockBankKeeper(ctrl)
	appKeeper := mocks.NewMockApplicationKeeper(ctrl)

	// Assert that the unstakedBalance was minted to the migration module account.
	bankKeeper.EXPECT().MintCoins(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(sdk.NewCoins(unstakedBalance)),
	).Return(nil).Times(1)

	// Assert that the unstakedBalance was transferred to the shannonDestAddr account.
	bankKeeper.EXPECT().SendCoinsFromModuleToAccount(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(shannonDestAccAddr),
		gomock.Eq(sdk.NewCoins(unstakedBalance)),
	).Return(nil).Times(1)

	// Simulate the application not existing.
	appKeeper.EXPECT().GetApplication(
		gomock.Any(),
		gomock.Eq(shannonDestAddr),
	).Return(apptypes.Application{}, false).AnyTimes()

	// Assert that the application was staked.
	appKeeper.EXPECT().SetApplication(
		gomock.Any(),
		gomock.Eq(expectedApp),
	).Return().Times(1)

	opts := []keepertest.MigrationKeeperOptionFn{
		keepertest.WithBankKeeper(bankKeeper),
		keepertest.WithApplicationKeeper(appKeeper),
	}

	k, ctx := keepertest.MigrationKeeper(t, opts...)
	ctx = ctx.WithBlockHeight(expectedClaimedAtHeight)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  sample.MorseAddressHex(),
		PublicKey:        morsePrivKey.PubKey().Bytes(),
		UnstakedBalance:  unstakedBalance,
		ApplicationStake: applicationStake,
		SupplierStake:    sdk.NewInt64Coin(volatile.DenomuPOKT, 0),
		// ShannonDestAddress: (intentionally omitted),
		// ClaimedAtHeight:    (intentionally omitted),
	}

	accountState := &migrationtypes.MorseAccountState{
		Accounts: []*migrationtypes.MorseClaimableAccount{
			morseClaimableAccount,
		},
	}
	accountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	// Import the MorseClaimableAccount.
	_, err = srv.ImportMorseClaimableAccounts(ctx, &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:             authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState:     *accountState,
		MorseAccountStateHash: accountStateHash,
	})
	require.NoError(t, err)

	// Claim the MorseClaimableAccount.
	msgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		morseClaimableAccount.GetMorseSrcAddress(),
		morsePrivKey,
		morseClaimableAccount.GetApplicationStake(),
		testServiceConfig,
	)
	require.NoError(t, err)

	msgClaimRes, err := srv.ClaimMorseApplication(ctx, msgClaim)
	require.NoError(t, err)

	// Construct and assert the expected response.
	expectedRes := &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         msgClaim.MorseSrcAddress,
		ClaimedApplicationStake: morseClaimableAccount.GetApplicationStake(),
		ClaimedBalance: morseClaimableAccount.GetUnstakedBalance().
			Add(morseClaimableAccount.GetSupplierStake()),
		ClaimedAtHeight: expectedClaimedAtHeight,
		ServiceId:       testServiceConfig.GetServiceId(),
	}
	require.Equal(t, expectedRes, msgClaimRes)

	// Assert that the persisted MorseClaimableAccount is updated.
	expectedMorseAccount := morseClaimableAccount
	expectedMorseAccount.ShannonDestAddress = shannonDestAddr
	expectedMorseAccount.ClaimedAtHeight = ctx.BlockHeight()
	foundMorseAccount, found := k.GetMorseClaimableAccount(ctx, msgClaim.MorseSrcAddress)
	require.True(t, found)
	require.Equal(t, *expectedMorseAccount, foundMorseAccount)

	// Assert that an event is emitted for each claim.
	expectedEvent := &migrationtypes.EventMorseAccountClaimed{
		ShannonDestAddress: msgClaim.ShannonDestAddress,
		MorseSrcAddress:    msgClaim.MorseSrcAddress,
		ClaimedBalance:     unstakedBalance,
		ClaimedAtHeight:    ctx.BlockHeight(),
	}
	claimEvents := events.FilterEvents[*migrationtypes.EventMorseAccountClaimed](t, ctx.EventManager().Events())
	require.Equal(t, 1, len(claimEvents))
	require.Equal(t, expectedEvent, claimEvents[0])

	// Reset the event manager to isolate events between claims.
	ctx = ctx.WithEventManager(sdk.NewEventManager())
}

func TestMsgServer_ClaimMorseApplication_SuccessExistingApplication(t *testing.T) {
	shannonDestAddr := sample.AccAddress()
	shannonDestAccAddr, err := sdk.AccAddressFromBech32(shannonDestAddr)
	require.NoError(t, err)

	claimableUnstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	claimableApplicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)

	expectedClaimedAtHeight := int64(10)
	initialAppServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc0"}
	initialApplicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 30)
	initialApp := apptypes.Application{
		Address:        shannonDestAddr,
		Stake:          &initialApplicationStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{initialAppServiceConfig},
	}

	expectedAppServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}
	expectedApplicationStake := initialApplicationStake.Add(claimableApplicationStake)
	expectedApp := apptypes.Application{
		Address:        shannonDestAddr,
		Stake:          &expectedApplicationStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{expectedAppServiceConfig},
	}

	ctrl := gomock.NewController(t)
	bankKeeper := mocks.NewMockBankKeeper(ctrl)
	appKeeper := mocks.NewMockApplicationKeeper(ctrl)

	// Assert that the claimableUnstakedBalance was minted to the migration module account.
	bankKeeper.EXPECT().MintCoins(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(sdk.NewCoins(claimableUnstakedBalance)),
	).Return(nil).Times(1)

	// Assert that the claimableUnstakedBalance was transferred to the shannonDestAddr account.
	bankKeeper.EXPECT().SendCoinsFromModuleToAccount(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(shannonDestAccAddr),
		gomock.Eq(sdk.NewCoins(claimableUnstakedBalance)),
	).Return(nil).Times(1)

	// Simulate an existing application.
	appKeeper.EXPECT().GetApplication(
		gomock.Any(),
		gomock.Eq(shannonDestAddr),
	).Return(initialApp, true).AnyTimes()

	// Assert that the application was updated.
	appKeeper.EXPECT().SetApplication(
		gomock.Any(),
		gomock.Eq(expectedApp),
	).Return().Times(1)

	opts := []keepertest.MigrationKeeperOptionFn{
		keepertest.WithBankKeeper(bankKeeper),
		keepertest.WithApplicationKeeper(appKeeper),
	}

	k, ctx := keepertest.MigrationKeeper(t, opts...)
	ctx = ctx.WithBlockHeight(expectedClaimedAtHeight)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  sample.MorseAddressHex(),
		PublicKey:        morsePrivKey.PubKey().Bytes(),
		UnstakedBalance:  claimableUnstakedBalance,
		ApplicationStake: claimableApplicationStake,
		SupplierStake:    sdk.NewInt64Coin(volatile.DenomuPOKT, 0),
		// ShannonDestAddress: (intentionally omitted),
		// ClaimedAtHeight:    (intentionally omitted),
	}

	accountState := &migrationtypes.MorseAccountState{
		Accounts: []*migrationtypes.MorseClaimableAccount{
			morseClaimableAccount,
		},
	}
	accountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	// Import the MorseClaimableAccount.
	_, err = srv.ImportMorseClaimableAccounts(ctx, &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:             authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState:     *accountState,
		MorseAccountStateHash: accountStateHash,
	})
	require.NoError(t, err)

	// Claim the MorseClaimableAccount.
	msgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		morseClaimableAccount.GetMorseSrcAddress(),
		morsePrivKey,
		morseClaimableAccount.GetApplicationStake(),
		expectedAppServiceConfig,
	)
	require.NoError(t, err)

	msgClaimRes, err := srv.ClaimMorseApplication(ctx, msgClaim)
	require.NoError(t, err)

	// Construct and assert the expected response.
	expectedRes := &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         msgClaim.MorseSrcAddress,
		ClaimedApplicationStake: expectedApplicationStake,
		ClaimedBalance:          claimableUnstakedBalance,
		ClaimedAtHeight:         expectedClaimedAtHeight,
		ServiceId:               expectedAppServiceConfig.GetServiceId(),
	}
	require.Equal(t, expectedRes, msgClaimRes)

	// Assert that the persisted MorseClaimableAccount is updated.
	expectedMorseAccount := morseClaimableAccount
	expectedMorseAccount.ShannonDestAddress = shannonDestAddr
	expectedMorseAccount.ClaimedAtHeight = ctx.BlockHeight()
	foundMorseAccount, found := k.GetMorseClaimableAccount(ctx, msgClaim.MorseSrcAddress)
	require.True(t, found)
	require.Equal(t, *expectedMorseAccount, foundMorseAccount)

	// Assert that an event is emitted for each claim.
	expectedEvent := &migrationtypes.EventMorseAccountClaimed{
		ShannonDestAddress: msgClaim.ShannonDestAddress,
		MorseSrcAddress:    msgClaim.MorseSrcAddress,
		ClaimedBalance: morseClaimableAccount.GetUnstakedBalance().
			Add(morseClaimableAccount.GetSupplierStake()),
		ClaimedAtHeight: ctx.BlockHeight(),
	}
	claimEvents := events.FilterEvents[*migrationtypes.EventMorseAccountClaimed](t, ctx.EventManager().Events())
	require.Equal(t, 1, len(claimEvents))
	require.Equal(t, expectedEvent, claimEvents[0])

	// Reset the event manager to isolate events between claims.
	ctx = ctx.WithEventManager(sdk.NewEventManager())
}

// TODO_IN_THIS_COMMIT: update - copy/pasta'd...
func TestMsgServer_ClaimMorseApplication_Error(t *testing.T) {
	claimableUnstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	claimableApplicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)
	expectedAppServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}

	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  sample.MorseAddressHex(),
		PublicKey:        morsePrivKey.PubKey().Bytes(),
		UnstakedBalance:  claimableUnstakedBalance,
		ApplicationStake: claimableApplicationStake,
		SupplierStake:    sdk.NewInt64Coin(volatile.DenomuPOKT, 0),
		// ShannonDestAddress: (intentionally omitted),
		// ClaimedAtHeight:    (intentionally omitted),
	}

	accountState := &migrationtypes.MorseAccountState{
		Accounts: []*migrationtypes.MorseClaimableAccount{
			morseClaimableAccount,
		},
	}
	accountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	_, err = srv.ImportMorseClaimableAccounts(ctx, &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:             authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState:     *accountState,
		MorseAccountStateHash: accountStateHash,
	})
	require.NoError(t, err)

	// Claim the MorseClaimableAccount with a random Shannon address.
	msgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
		sample.AccAddress(),
		accountState.Accounts[0].GetMorseSrcAddress(),
		morsePrivKey,
		claimableApplicationStake,
		expectedAppServiceConfig,
	)
	require.NoError(t, err)

	t.Run("invalid claim msg", func(t *testing.T) {
		// Copy the message and set the morse signature to an empty string.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSignature = ""

		expectedErr := status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"morseSignature is empty",
			).Error(),
		)

		_, err := srv.ClaimMorseApplication(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account not found", func(t *testing.T) {
		// Copy the message and set the morse src address to a valid but incorrect address.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSrcAddress = sample.MorseAddressHex()

		expectedErr := status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"no morse claimable account exists with address %q",
				invalidMsgClaim.GetMorseSrcAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseApplication(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-zero claimed_at_height)", func(t *testing.T) {
		// Set the claimed at height BUT NOT the Shannon destination address.
		morseClaimableAccount := *accountState.Accounts[0]
		morseClaimableAccount.ClaimedAtHeight = 10
		k.SetMorseClaimableAccount(ctx, morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				10,
				accountState.Accounts[0].GetShannonDestAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseApplication(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-empty shannon_dest_address)", func(t *testing.T) {
		// Set the Shannon destination address BUT NOT the claimed at height.
		morseClaimableAccount := *accountState.Accounts[0]
		morseClaimableAccount.ClaimedAtHeight = 0
		morseClaimableAccount.ShannonDestAddress = sample.AccAddress()
		k.SetMorseClaimableAccount(ctx, morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				0,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)

		_, err := srv.ClaimMorseApplication(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})
}
