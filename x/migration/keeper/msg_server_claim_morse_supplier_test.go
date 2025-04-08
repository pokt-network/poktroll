package keeper_test

import (
	"strconv"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

var (
	// Prevent strconv unused error
	_ = strconv.IntSize

	// Shared testing supplier service config
	testSupplierServices = []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: "svc1",
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://test.example:1234",
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            sample.AccAddress(),
					RevSharePercentage: 100,
				},
			},
		},
	}
)

func TestMsgServer_ClaimMorseSupplier_SuccessNewSupplier(t *testing.T) {
	shannonDestAddr := sample.AccAddress()
	shannonDestAccAddr, err := cosmostypes.AccAddressFromBech32(shannonDestAddr)
	require.NoError(t, err)

	claimCommitHeight := int64(10)
	unstakedBalance := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)
	supplierStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 200)
	expectedMintCoin := unstakedBalance.Add(supplierStake)
	expectedClaimedUnstakedTokens := expectedMintCoin.Sub(supplierStake)
	expectedMsgStakeSupplier := &suppliertypes.MsgStakeSupplier{
		Signer:          shannonDestAddr,
		OwnerAddress:    shannonDestAddr,
		OperatorAddress: shannonDestAddr,
		Stake:           &supplierStake,
		Services:        testSupplierServices,
	}
	expectedSupplier := sharedtypes.Supplier{
		OwnerAddress:    shannonDestAddr,
		OperatorAddress: shannonDestAddr,
		Stake:           &supplierStake,
		Services:        testSupplierServices,
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
			{
				Services:             testSupplierServices,
				EffectiveBlockHeight: 1,
			},
		},
		UnstakeSessionEndHeight: 0,
	}

	ctrl := gomock.NewController(t)
	bankKeeper := mocks.NewMockBankKeeper(ctrl)
	supplierKeeper := mocks.NewMockSupplierKeeper(ctrl)

	// Assert that the unstakedBalance was minted to the migration module account.
	bankKeeper.EXPECT().MintCoins(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(cosmostypes.NewCoins(expectedMintCoin)),
	).Return(nil).Times(1)

	// Assert that the unstakedBalance was transferred to the shannonDestAddr account.
	bankKeeper.EXPECT().SendCoinsFromModuleToAccount(
		gomock.Any(),
		gomock.Eq(migrationtypes.ModuleName),
		gomock.Eq(shannonDestAccAddr),
		gomock.Eq(cosmostypes.NewCoins(expectedMintCoin)),
	).Return(nil).Times(1)

	// Simulate the application not existing.
	supplierKeeper.EXPECT().GetSupplier(
		gomock.Any(),
		gomock.Eq(shannonDestAddr),
	).Return(sharedtypes.Supplier{}, false).AnyTimes()

	// Assert that the application was staked.
	supplierKeeper.EXPECT().StakeSupplier(
		gomock.Any(),
		gomock.Any(),
		gomock.Eq(expectedMsgStakeSupplier),
	).Return(&expectedSupplier, nil).Times(1)

	opts := []keepertest.MigrationKeeperOptionFn{
		keepertest.WithBankKeeper(bankKeeper),
		keepertest.WithSupplierKeeper(supplierKeeper),
	}

	k, ctx := keepertest.MigrationKeeper(t, opts...)
	ctx = ctx.WithBlockHeight(claimCommitHeight)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
		UnstakedBalance:  unstakedBalance,
		ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
		SupplierStake:    supplierStake,
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
	msgClaim, err := migrationtypes.NewMsgClaimMorseSupplier(
		shannonDestAddr,
		shannonDestAddr,
		morsePrivKey,
		testSupplierServices,
	)
	require.NoError(t, err)

	msgClaimRes, err := srv.ClaimMorseSupplier(ctx, msgClaim)
	require.NoError(t, err)

	// Construct and assert the expected response.
	sharedParams := sharedtypes.DefaultParams()
	expectedSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, ctx.BlockHeight())
	expectedRes := &migrationtypes.MsgClaimMorseSupplierResponse{
		MorseSrcAddress:      msgClaim.MorseSrcAddress,
		ClaimedSupplierStake: morseClaimableAccount.GetSupplierStake(),
		ClaimedBalance: expectedClaimedUnstakedTokens.
			Add(morseClaimableAccount.GetApplicationStake()),
		SessionEndHeight: expectedSessionEndHeight,
		Supplier:         &expectedSupplier,
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
	expectedEvent := &migrationtypes.EventMorseSupplierClaimed{
		MorseSrcAddress:      msgClaim.MorseSrcAddress,
		ClaimedBalance:       expectedClaimedUnstakedTokens,
		ClaimedSupplierStake: supplierStake,
		SessionEndHeight:     expectedSessionEndHeight,
		Supplier:             &expectedSupplier,
	}
	claimEvents := events.FilterEvents[*migrationtypes.EventMorseSupplierClaimed](t, ctx.EventManager().Events())
	require.Equal(t, 1, len(claimEvents))
	require.Equal(t, expectedEvent, claimEvents[0])
}

func TestMsgServer_ClaimMorseSupplier_Error(t *testing.T) {
	claimableUnstakedBalance := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)
	claimableSupplierStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 200)

	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.GenMorsePrivateKey(0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
		UnstakedBalance:  claimableUnstakedBalance,
		ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
		SupplierStake:    claimableSupplierStake,
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

	// Claim the MorseClaimableAccount with random Shannon owner & operator addresses.
	msgClaim, err := migrationtypes.NewMsgClaimMorseSupplier(
		sample.AccAddress(),
		sample.AccAddress(),
		morsePrivKey,
		testSupplierServices,
	)
	require.NoError(t, err)

	wrongMorsePrivKey := testmigration.GenMorsePrivateKey(99)

	t.Run("invalid claim msg", func(t *testing.T) {
		invalidClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServices,
		)
		require.NoError(t, err)

		// Set the Morse signature to nil to simulate a missing signature.
		invalidClaimMsg.MorseSignature = nil

		expectedErr := status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseSignature.Wrapf(
				"invalid morse signature length; expected %d, got %d",
				migrationtypes.MorseSignatureLengthBytes, 0,
			).Error(),
		)

		_, err = srv.ClaimMorseSupplier(ctx, invalidClaimMsg)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account not found", func(t *testing.T) {
		invalidMsgClaim, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			wrongMorsePrivKey,
			testSupplierServices,
		)
		require.NoError(t, err)

		expectedErr := status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"no morse claimable account exists with address %q",
				invalidMsgClaim.GetMorseSrcAddress(),
			).Error(),
		)

		_, err = srv.ClaimMorseSupplier(ctx, invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-zero claimed_at_height)", func(t *testing.T) {
		// Set the claimed at height BUT NOT the Shannon destination address.
		morseClaimableAccount.ClaimedAtHeight = 10
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				10,
				accountState.Accounts[0].GetShannonDestAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseSupplier(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-empty shannon_dest_address)", func(t *testing.T) {
		// Set the Shannon destination address BUT NOT the claimed at height.
		morseClaimableAccount.ClaimedAtHeight = 0
		morseClaimableAccount.ShannonDestAddress = sample.AccAddress()
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				0,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)

		_, err := srv.ClaimMorseSupplier(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("morse account not staked as any actor", func(t *testing.T) {
		nonSupplierMorseClaimableAccount := migrationtypes.MorseClaimableAccount{
			MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
			UnstakedBalance:  claimableUnstakedBalance,
			SupplierStake:    cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
			ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
		}
		k.SetMorseClaimableAccount(ctx, nonSupplierMorseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is not staked as an supplier or application, please use `pocketd migrate claim-account` instead",
				nonSupplierMorseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)

		// Claim the MorseClaimableAccount with random Shannon owner & operator addresses.
		msgClaim, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServices,
		)
		require.NoError(t, err)

		_, err = srv.ClaimMorseSupplier(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("morse account staked as an application", func(t *testing.T) {
		nonSupplierMorseClaimableAccount := migrationtypes.MorseClaimableAccount{
			MorseSrcAddress:  morsePrivKey.PubKey().Address().String(),
			UnstakedBalance:  claimableUnstakedBalance,
			SupplierStake:    cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
			ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100),
		}
		k.SetMorseClaimableAccount(ctx, nonSupplierMorseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is staked as an application, please use `pocketd migrate claim-application` instead",
				nonSupplierMorseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)

		// Claim the MorseClaimableAccount with random Shannon owner & operator addresses.
		msgClaim, err := migrationtypes.NewMsgClaimMorseSupplier(
			sample.AccAddress(),
			sample.AccAddress(),
			morsePrivKey,
			testSupplierServices,
		)
		require.NoError(t, err)

		_, err = srv.ClaimMorseSupplier(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})
}
