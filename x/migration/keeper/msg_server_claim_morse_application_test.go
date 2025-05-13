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
var (
	_                    = strconv.IntSize
	testAppServiceConfig = sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}
)

func TestMsgServer_ClaimMorseApplication_SuccessNewApplication(t *testing.T) {
	shannonDestAddr := sample.AccAddress()
	shannonDestAccAddr, err := sdk.AccAddressFromBech32(shannonDestAddr)
	require.NoError(t, err)

	expectedClaimedAtHeight := int64(10)
	unstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	applicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)
	expectedMintCoin := unstakedBalance.Add(applicationStake)
	expectedClaimedUnstakedTokens := expectedMintCoin.Sub(applicationStake)
	expectedMsgStakeApp := &apptypes.MsgStakeApplication{
		Address:  shannonDestAddr,
		Stake:    &applicationStake,
		Services: []*sharedtypes.ApplicationServiceConfig{&testAppServiceConfig},
	}
	expectedApp := apptypes.Application{
		Address:                   shannonDestAddr,
		Stake:                     &applicationStake,
		ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{&testAppServiceConfig},
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
	}

	ctrl := gomock.NewController(t)
	bankKeeper := mocks.NewMockBankKeeper(ctrl)
	appKeeper := mocks.NewMockApplicationKeeper(ctrl)

	// Assert that the unstakedBalance was minted to the migration module account.
	bankKeeper.EXPECT().MintCoins(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(sdk.NewCoins(expectedMintCoin)),
	).Return(nil).Times(1)

	// Assert that the unstakedBalance was transferred to the shannonDestAddr account.
	bankKeeper.EXPECT().SendCoinsFromModuleToAccount(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(shannonDestAccAddr),
		gomock.Eq(sdk.NewCoins(expectedMintCoin)),
	).Return(nil).Times(1)

	// Simulate the application not existing.
	appKeeper.EXPECT().GetApplication(
		gomock.Any(),
		gomock.Eq(shannonDestAddr),
	).Return(apptypes.Application{}, false).AnyTimes()

	// Assert that the application was staked.
	appKeeper.EXPECT().StakeApplication(
		gomock.Any(),
		gomock.Any(),
		gomock.Eq(expectedMsgStakeApp),
	).Return(&expectedApp, nil).Times(1)

	opts := []keepertest.MigrationKeeperOptionFn{
		keepertest.WithBankKeeper(bankKeeper),
		keepertest.WithApplicationKeeper(appKeeper),
	}

	k, ctx := keepertest.MigrationKeeper(t, opts...)
	ctx = ctx.WithBlockHeight(expectedClaimedAtHeight)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
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
		morsePrivKey,
		&testAppServiceConfig,
		sample.AccAddress(),
	)
	require.NoError(t, err)

	msgClaimRes, err := srv.ClaimMorseApplication(ctx, msgClaim)
	require.NoError(t, err)

	// Construct and assert the expected response.
	sharedParams := sharedtypes.DefaultParams()
	expectedSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, ctx.BlockHeight())
	expectedRes := &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         msgClaim.GetMorseSrcAddress(),
		ClaimedApplicationStake: morseClaimableAccount.GetApplicationStake(),
		ClaimedBalance: expectedClaimedUnstakedTokens.
			Add(morseClaimableAccount.GetSupplierStake()),
		SessionEndHeight: expectedSessionEndHeight,
		Application:      &expectedApp,
	}
	require.Equal(t, expectedRes, msgClaimRes)

	// Assert that the persisted MorseClaimableAccount is updated.
	expectedMorseAccount := morseClaimableAccount
	expectedMorseAccount.ShannonDestAddress = shannonDestAddr
	expectedMorseAccount.ClaimedAtHeight = ctx.BlockHeight()
	foundMorseAccount, found := k.GetMorseClaimableAccount(ctx, msgClaim.GetMorseSrcAddress())
	require.True(t, found)
	require.Equal(t, *expectedMorseAccount, foundMorseAccount)

	// Assert that an event is emitted for each claim.
	expectedEvent := &migrationtypes.EventMorseApplicationClaimed{
		MorseSrcAddress:         msgClaim.GetMorseSrcAddress(),
		ClaimedBalance:          expectedClaimedUnstakedTokens,
		ClaimedApplicationStake: applicationStake,
		SessionEndHeight:        expectedSessionEndHeight,
		Application:             &expectedApp,
	}
	claimEvents := events.FilterEvents[*migrationtypes.EventMorseApplicationClaimed](t, ctx.EventManager().Events())
	require.Equal(t, 1, len(claimEvents))
	require.Equal(t, expectedEvent, claimEvents[0])
}

func TestMsgServer_ClaimMorseApplication_Error(t *testing.T) {
	claimableUnstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	claimableApplicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)
	expectedAppServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}

	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
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
		morsePrivKey,
		expectedAppServiceConfig,
		sample.AccAddress(),
	)
	require.NoError(t, err)

	// Generate a Morse private key for an account which is not in the Morse account state.
	nonExistentMorsePrivKey := testmigration.GenMorsePrivateKey(99)

	t.Run("invalid claim msg", func(t *testing.T) {
		invalidMsgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
			msgClaim.GetShannonDestAddress(),
			morsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		invalidMsgClaim.MorseSignature = nil
		expectedErr := status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseSignature.Wrapf(
				"invalid morse signature length; expected %d, got %d",
				migrationtypes.MorseSignatureLengthBytes, 0,
			).Error(),
		)

		_, err = srv.ClaimMorseApplication(ctx, invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account not found", func(t *testing.T) {
		invalidMsgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
			msgClaim.GetShannonDestAddress(),
			nonExistentMorsePrivKey,
			&testAppServiceConfig,
			sample.AccAddress(),
		)
		require.NoError(t, err)

		expectedErr := status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"no morse claimable account exists with address %q",
				invalidMsgClaim.GetMorseSrcAddress(),
			).Error(),
		)

		_, err = srv.ClaimMorseApplication(ctx, invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-zero claimed_at_height)", func(t *testing.T) {
		// Set the claimed at height BUT NOT the Shannon destination address.
		morseClaimableAccount.ClaimedAtHeight = 10
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

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
		morseClaimableAccount.ClaimedAtHeight = 0
		morseClaimableAccount.ShannonDestAddress = sample.AccAddress()
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

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
