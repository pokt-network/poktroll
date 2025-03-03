package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
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
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

var zeroUpokt = sdk.NewInt64Coin(volatile.DenomuPOKT, 0)

func init() {
	// DEV_NOTE: Due to an optimization in big.Int, strict equality checking MAY fail with 0 amount coins.
	// To work around this, we can initialize the bit.Int with a non-zero value and then set it to zero via arithmetic.
	zeroUpokt.Amount = math.NewInt(1).SubRaw(1)
}

func TestMsgServer_ClaimMorseGateway_SuccessNewGateway(t *testing.T) {
	shannonDestAddr := sample.AccAddress()
	shannonDestAccAddr, err := sdk.AccAddressFromBech32(shannonDestAddr)
	require.NoError(t, err)

	expectedClaimedAtHeight := int64(10)
	unstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	gatewayStakeToClaim := unstakedBalance
	expectedMintCoin := unstakedBalance
	expectedClaimedUnstakedTokens := expectedMintCoin.Sub(gatewayStakeToClaim)
	if expectedClaimedUnstakedTokens.IsZero() {
		expectedClaimedUnstakedTokens = zeroUpokt
	}

	expectedMsgStakeGateway := &gatewaytypes.MsgStakeGateway{
		Address: shannonDestAddr,
		Stake:   &gatewayStakeToClaim,
	}
	expectedGateway := gatewaytypes.Gateway{
		Address: shannonDestAddr,
		Stake:   &gatewayStakeToClaim,
	}

	ctrl := gomock.NewController(t)
	bankKeeper := mocks.NewMockBankKeeper(ctrl)
	gatewayKeeper := mocks.NewMockGatewayKeeper(ctrl)

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

	// Simulate the gateay not existing.
	gatewayKeeper.EXPECT().GetGateway(
		gomock.Any(),
		gomock.Eq(shannonDestAddr),
	).Return(gatewaytypes.Gateway{}, false).AnyTimes()

	// Assert that the application was staked.
	gatewayKeeper.EXPECT().StakeGateway(
		gomock.Any(),
		gomock.Any(),
		gomock.Eq(expectedMsgStakeGateway),
	).Return(&expectedGateway, nil).Times(1)

	opts := []keepertest.MigrationKeeperOptionFn{
		keepertest.WithBankKeeper(bankKeeper),
		keepertest.WithGatewayKeeper(gatewayKeeper),
	}

	k, ctx := keepertest.MigrationKeeper(t, opts...)
	ctx = ctx.WithBlockHeight(expectedClaimedAtHeight)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  sample.MorseAddressHex(),
		PublicKey:        morsePrivKey.PubKey().Bytes(),
		UnstakedBalance:  unstakedBalance,
		ApplicationStake: zeroUpokt,
		SupplierStake:    zeroUpokt,
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
	msgClaim, err := migrationtypes.NewMsgClaimMorseGateway(
		shannonDestAddr,
		morseClaimableAccount.GetMorseSrcAddress(),
		morsePrivKey,
		gatewayStakeToClaim,
	)
	require.NoError(t, err)

	msgClaimRes, err := srv.ClaimMorseGateway(ctx, msgClaim)
	require.NoError(t, err)

	// Construct and assert the expected response.
	expectedRes := &migrationtypes.MsgClaimMorseGatewayResponse{
		MorseSrcAddress:     msgClaim.MorseSrcAddress,
		ClaimedGatewayStake: gatewayStakeToClaim,
		ClaimedBalance:      expectedClaimedUnstakedTokens,
		ClaimedAtHeight:     expectedClaimedAtHeight,
		Gateway:             &expectedGateway,
	}
	require.Equal(t, expectedRes, msgClaimRes)

	// Assert that the persisted MorseClaimableAccount is updated.
	expectedMorseAccount := morseClaimableAccount
	expectedMorseAccount.ShannonDestAddress = shannonDestAddr
	expectedMorseAccount.ClaimedAtHeight = ctx.BlockHeight()
	foundMorseAccount, found := k.GetMorseClaimableAccount(ctx, msgClaim.MorseSrcAddress)
	require.True(t, found)

	if foundMorseAccount.SupplierStake.IsZero() {
		foundMorseAccount.SupplierStake = zeroUpokt
	}
	if foundMorseAccount.ApplicationStake.IsZero() {
		foundMorseAccount.ApplicationStake = zeroUpokt
	}
	require.Equal(t, *expectedMorseAccount, foundMorseAccount)

	// DEV_NOTE: Due the same optimization in big.Int as above, the following assertion will also fail otherwise.
	if expectedClaimedUnstakedTokens.IsZero() {
		expectedClaimedUnstakedTokens.Amount = math.NewInt(0)
	}

	// Assert that an event is emitted for each claim.
	expectedEvent := &migrationtypes.EventMorseGatewayClaimed{
		ShannonDestAddress:  msgClaim.ShannonDestAddress,
		MorseSrcAddress:     msgClaim.MorseSrcAddress,
		ClaimedBalance:      expectedClaimedUnstakedTokens,
		ClaimedGatewayStake: gatewayStakeToClaim,
		ClaimedAtHeight:     ctx.BlockHeight(),
		Gateway:             &expectedGateway,
	}
	claimEvents := events.FilterEvents[*migrationtypes.EventMorseGatewayClaimed](t, ctx.EventManager().Events())
	require.Equal(t, 1, len(claimEvents))
	require.Equal(t, expectedEvent, claimEvents[0])
}

func TestMsgServer_ClaimMorseGateway_Error(t *testing.T) {
	claimableUnstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	gatewayStakeToClaim := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)

	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)
	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  sample.MorseAddressHex(),
		PublicKey:        morsePrivKey.PubKey().Bytes(),
		UnstakedBalance:  claimableUnstakedBalance,
		ApplicationStake: zeroUpokt,
		SupplierStake:    zeroUpokt,
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
	msgClaim, err := migrationtypes.NewMsgClaimMorseGateway(
		sample.AccAddress(),
		accountState.Accounts[0].GetMorseSrcAddress(),
		morsePrivKey,
		gatewayStakeToClaim,
	)
	require.NoError(t, err)

	t.Run("invalid claim msg", func(t *testing.T) {
		// Copy the message and set the morse signature to nil.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSignature = nil

		expectedErr := status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"invalid morse signature length; expected %d, got %d",
				migrationtypes.MorseSignatureLengthBytes, 0,
			).Error(),
		)

		_, err := srv.ClaimMorseGateway(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account not found", func(t *testing.T) {
		// Copy the message and set the morse src address to a valid but incorrect address.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSrcAddress = sample.MorseAddressHex()

		expectedErr := status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"no morse claimable account exists with address %q",
				invalidMsgClaim.GetMorseSrcAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseGateway(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-zero claimed_at_height)", func(t *testing.T) {
		// Set the claimed at height BUT NOT the Shannon destination address.
		morseClaimableAccount.ClaimedAtHeight = 10
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				10,
				accountState.Accounts[0].GetShannonDestAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseGateway(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-empty shannon_dest_address)", func(t *testing.T) {
		// Set the Shannon destination address BUT NOT the claimed at height.
		morseClaimableAccount.ClaimedAtHeight = 0
		morseClaimableAccount.ShannonDestAddress = sample.AccAddress()
		k.SetMorseClaimableAccount(ctx, *morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				0,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)

		_, err := srv.ClaimMorseGateway(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})
}
