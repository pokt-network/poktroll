package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMsgServer_ClaimMorseApplication_Success(t *testing.T) {
	testServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}

	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// TODO_IN_THIS_COMMIT: comment...
	unstakedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1000)
	applicationStake := sdk.NewInt64Coin(volatile.DenomuPOKT, 200)
	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)

	accountState := &migrationtypes.MorseAccountState{
		Accounts: []*migrationtypes.MorseClaimableAccount{
			{
				MorseSrcAddress:  sample.MorseAddressHex(),
				PublicKey:        morsePrivKey.PubKey().Bytes(),
				UnstakedBalance:  unstakedBalance,
				ApplicationStake: applicationStake,
				SupplierStake:    sdk.NewInt64Coin(volatile.DenomuPOKT, 0),
				// ShannonDestAddress: (intentionally omitted),
				// ClaimedAtHeight:    (intentionally omitted),
			},
		},
	}
	accountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	// TODO_IN_THIS_COMMIT: comment... Import ...
	_, err = srv.ImportMorseClaimableAccounts(ctx, &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:             authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState:     *accountState,
		MorseAccountStateHash: accountStateHash,
	})
	require.NoError(t, err)

	// Claim each MorseClaimableAccount as an application.
	for morseAccountIdx, morseAccount := range accountState.Accounts {
		// Generate the corresponding morse private key using the account slice index as a seed.
		morsePrivKey := testmigration.NewMorsePrivateKey(t, uint64(morseAccountIdx))

		// Claim the MorseClaimableAccount.
		msgClaim, err := migrationtypes.NewMsgClaimMorseApplication(
			sample.AccAddress(),
			morseAccount.GetMorseSrcAddress(),
			morsePrivKey,
			morseAccount.GetApplicationStake(),
			testServiceConfig,
		)
		require.NoError(t, err)

		msgClaimRes, err := srv.ClaimMorseApplication(ctx, msgClaim)
		require.NoError(t, err)

		// Construct and assert the expected response.
		expectedRes := &migrationtypes.MsgClaimMorseApplicationResponse{
			MorseSrcAddress:         msgClaim.MorseSrcAddress,
			ClaimedApplicationStake: morseAccount.GetApplicationStake(),
			ClaimedBalance: morseAccount.GetUnstakedBalance().
				Add(morseAccount.GetSupplierStake()),
			ClaimedAtHeight: ctx.BlockHeight(),
			ServiceId:       testServiceConfig.GetServiceId(),
		}
		require.Equal(t, expectedRes, msgClaimRes)

		// Assert that the persisted MorseClaimableAccount is updated.
		expectedMorseAccount := morseAccount
		expectedMorseAccount.ClaimedAtHeight = ctx.BlockHeight()
		foundMorseAccount, found := k.GetMorseClaimableAccount(ctx, msgClaim.MorseSrcAddress)
		require.True(t, found)
		require.Equal(t, *expectedMorseAccount, foundMorseAccount)

		// Assert that an event is emitted for each claim.
		expectedEvent := &migrationtypes.EventMorseAccountClaimed{
			ShannonDestAddress: msgClaim.ShannonDestAddress,
			MorseSrcAddress:    msgClaim.MorseSrcAddress,
			ClaimedBalance: morseAccount.GetUnstakedBalance().
				Add(morseAccount.GetSupplierStake()),
			ClaimedAtHeight: ctx.BlockHeight(),
		}
		claimEvents := events.FilterEvents[*migrationtypes.EventMorseAccountClaimed](t, ctx.EventManager().Events())
		require.Equal(t, 1, len(claimEvents))
		require.Equal(t, expectedEvent, claimEvents[0])

		// Assert that the application was staked.
		//foundApp, found := k.GetApplication(ctx, msgClaim.ShannonDestAddress)
		//require.True(t, found)
		//require.Equal(t, msgClaim.Stake, foundApp.)
		//
		// TODO_IN_THIS_COMMIT: resume here...
		// TODO_IN_THIS_COMMIT: resume here...
		// TODO_IN_THIS_COMMIT: resume here...
		// TODO_IN_THIS_COMMIT: resume here...

		// Reset the event manager to isolate events between claims.
		ctx = ctx.WithEventManager(sdk.NewEventManager())
	}
}

func TestMsgServer_ClaimMorseApplication_Error(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate and import a Morse claimable account.
	numAccounts := 1
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)
	accountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	_, err = srv.ImportMorseClaimableAccounts(ctx, &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:             authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState:     *accountState,
		MorseAccountStateHash: accountStateHash,
	})
	require.NoError(t, err)

	// Generate the corresponding morse private key using the account slice index as a seed.
	morsePrivKey := testmigration.NewMorsePrivateKey(t, 0)

	// Claim the MorseClaimableAccount with a random Shannon address.
	msgClaim, err := migrationtypes.NewMsgClaimMorseAccount(
		sample.AccAddress(),
		accountState.Accounts[0].GetMorseSrcAddress(),
		morsePrivKey,
	)
	require.NoError(t, err)

	t.Run("invalid claim msg", func(t *testing.T) {
		// Copy the message and set the morse signature to an empty string.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSignature = ""

		expectedErr := status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morseSignature is empty",
			).Error(),
		)

		_, err := srv.ClaimMorseAccount(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account not found", func(t *testing.T) {
		// Copy the message and set the morse src address to a valid but incorrect address.
		invalidMsgClaim := *msgClaim
		invalidMsgClaim.MorseSrcAddress = sample.MorseAddressHex()

		expectedErr := status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"no morse claimable account exists with address %q",
				invalidMsgClaim.GetMorseSrcAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseAccount(ctx, &invalidMsgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("account already claimed (non-zero claimed_at_height)", func(t *testing.T) {
		// Set the claimed at height BUT NOT the Shannon destination address.
		morseClaimableAccount := *accountState.Accounts[0]
		morseClaimableAccount.ClaimedAtHeight = 10
		k.SetMorseClaimableAccount(ctx, morseClaimableAccount)

		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				10,
				accountState.Accounts[0].GetShannonDestAddress(),
			).Error(),
		)

		_, err := srv.ClaimMorseAccount(ctx, msgClaim)
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
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				accountState.Accounts[0].GetMorseSrcAddress(),
				0,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)

		_, err := srv.ClaimMorseAccount(ctx, msgClaim)
		require.EqualError(t, err, expectedErr.Error())
	})
}
