package migration

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	oneDayFromNow = time.Now().Add(24 * time.Hour)
	oneDayAgo     = time.Now().Add(-24 * time.Hour)
)

// TestClaimMorseApplication exercises claiming of a MorseClaimableAccount as a new staked application.
func (s *MigrationModuleTestSuite) TestClaimMorseNewApplication() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.AllApplicationMorseAccountActorType)
	s.ImportMorseClaimableAccounts(s.T())

	for morseAccountIdx, morseClaimableAccount := range s.GetAccountState(s.T()).Accounts {
		testDesc := fmt.Sprintf("morse account %d", morseAccountIdx)
		s.Run(testDesc, func() {
			shannonDestAddr := sample.AccAddress()
			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as a new application.
			morseSrcAddr, claimAppRes := s.ClaimMorseApplication(
				s.T(), uint64(morseAccountIdx),
				shannonDestAddr,
				s.appServiceConfig,
				sample.AccAddress(),
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[morseAccountIdx]
			expectedStake := morseClaimableAccount.GetApplicationStake()

			// Assert that the claim msg response is correct.
			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
				Add(expectedMorseClaimableAccount.GetApplicationStake()).
				Add(expectedMorseClaimableAccount.GetSupplierStake()).
				Sub(expectedStake)

			expectedApp := apptypes.Application{
				Address:        shannonDestAddr,
				Stake:          &expectedStake,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			}
			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{}
			s.Equal(expectedClaimApplicationRes, claimAppRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			s.Equal(expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			s.NoError(err)
			s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the application was staked.
			appClient := s.AppSuite.GetAppQueryClient(s.T())
			app, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedApp, app)
		})
	}
}

// TestClaimMorseApplication exercises claiming of a MorseClaimableAccount as an existing staked application.
func (s *MigrationModuleTestSuite) TestClaimMorseExistingApplication() {
	// Generate and import Morse claimable accounts.
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.AllApplicationMorseAccountActorType)
	s.ImportMorseClaimableAccounts(s.T())

	for morseAccountIdx := range s.GetAccountState(s.T()).Accounts {
		testDesc := fmt.Sprintf("morse account %d", morseAccountIdx)
		s.Run(testDesc, func() {
			// Stake an initial application.
			shannonDestAddr := sample.AccAddress()
			shannonDestAccAddr := cosmostypes.MustAccAddressFromBech32(shannonDestAddr)

			initialAppStake := &s.minStake
			s.FundAddress(s.T(), shannonDestAccAddr, initialAppStake.Amount.Int64())
			s.AppSuite.StakeApp(s.T(), shannonDestAddr, initialAppStake.Amount.Int64(), []string{"nosvc"})

			// Assert that the initial application is staked.
			appClient := s.AppSuite.GetAppQueryClient(s.T())
			foundApp, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(shannonDestAddr, foundApp.Address)

			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as an existing application.
			morseSrcAddr, claimAppRes := s.ClaimMorseApplication(
				s.T(), uint64(morseAccountIdx),
				shannonDestAddr,
				s.appServiceConfig,
				sample.AccAddress(),
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[morseAccountIdx]
			expectedClaimedStake := expectedMorseClaimableAccount.GetApplicationStake()
			expectedFinalStake := initialAppStake.Add(expectedClaimedStake)
			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance()

			// Assert that the claim msg response is correct.
			expectedApp := apptypes.Application{
				Address:        shannonDestAddr,
				Stake:          &expectedFinalStake,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			}
			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{}
			s.Equal(expectedClaimApplicationRes, claimAppRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			s.Equal(expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			s.NoError(err)
			s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the application was updated.
			app, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedApp, app)
		})
	}
}

func (s *MigrationModuleTestSuite) TestClaimMorseApplication_BelowMinStake() {
	// Set the min app stake param to just above the application stake amount.
	minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, testmigration.GenMorseApplicationStakeAmount(uint64(0))+1)
	s.ResetTestApp(1, minStake)
	s.GenerateMorseAccountState(s.T(), 1, testmigration.AllApplicationMorseAccountActorType)
	s.ImportMorseClaimableAccounts(s.T())

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.GenMorsePrivateKey(0)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		morsePrivateKey,
		s.appServiceConfig,
		sample.AccAddress(),
	)
	s.NoError(err)

	// Claim a Morse claimable account.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	s.NoError(err)

	// Assert that the MorseClaimableAccount was updated on-chain.
	lastCommittedHeight := s.SdkCtx().BlockHeight() - 1
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
	s.Equal(lastCommittedHeight, morseClaimableAccount.GetClaimedAtHeight())
	s.Equal(shannonDestAddr, morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance increased by the
	// MorseClaimableAccount (unstaked balance + application stake).
	expectedBalance := morseClaimableAccount.TotalTokens()
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(expectedBalance.Amount.Int64(), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	s.NoError(err)
	s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the application was NOT staked.
	appClient := s.AppSuite.GetAppQueryClient(s.T())
	_, err = appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
	require.EqualError(s.T(), err, status.Error(
		codes.NotFound,
		apptypes.ErrAppNotFound.Wrapf(
			"app address: %s",
			shannonDestAddr,
		).Error(),
	).Error())
}

func (s *MigrationModuleTestSuite) TestMsgClaimMorseApplication_Unbonding() {
	s.T().Skip("TODO_URGENT(@red-0ne): Skipping this test to unblock community and exchanges during the migration. See #1436.")

	// Configure fixtures to generate Morse applications which have begun unbonding on Morse:
	// - 1 whose unbonding period HAS NOT yet elapsed
	// - 1 whose unbonding period HAS elapsed
	unbondingActorsOpt := testmigration.WithUnbondingActors(testmigration.UnbondingActorsConfig{
		// Number of applications to generate as having begun unbonding on Morse
		NumApplicationsUnbondingBegan: 1,

		// Number of applications to generate as having unbonded on Morse while waiting to be claimed
		NumApplicationsUnbondingEnded: 1,
	})

	// Configure fixtures to generate Morse application balances:
	// - Staked balance is 1upokt above the minimum stake (101upokt)
	// - Unstaked balance is 420upokt ✌️
	appStakesFnOpt := testmigration.WithApplicationStakesFn(func(
		_, _ uint64,
		_ testmigration.MorseApplicationActorType,
		_ *migrationtypes.MorseApplication,
	) (staked, unstaked *cosmostypes.Coin) {
		staked, unstaked = new(cosmostypes.Coin), new(cosmostypes.Coin)
		*staked = s.minStake.Add(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1))
		*unstaked = cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 420)
		return staked, unstaked
	})

	// Configure fixtures to generate unstaking times which correspond to the
	// application actor type (i.e. unbonding or unbonded).
	unstakingTimeConfig := testmigration.WithUnstakingTime(testmigration.UnstakingTimeConfig{
		ApplicationUnstakingTimeFn: func(
			_, _ uint64,
			actorType testmigration.MorseApplicationActorType,
			_ *migrationtypes.MorseApplication,
		) time.Time {
			switch actorType {

			case testmigration.MorseUnbondingApplication:
				return oneDayFromNow

			case testmigration.MorseUnbondedApplication:

				return oneDayAgo
			default:
				// Don't set unstaking time for any other application actor types.
				return time.Time{}
			}
		},
	})

	// Generate and import Morse claimable accounts.
	fixtures, err := testmigration.NewMorseFixtures(
		unbondingActorsOpt,
		appStakesFnOpt,
		unstakingTimeConfig,
	)
	s.NoError(err)

	s.SetMorseAccountState(s.T(), fixtures.GetMorseAccountState())
	_, err = s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	// DEV_NOTE: The accounts/actors are generated in the order they are defined in the UnbondingActorsConfig struct.
	unbondingAppFixture := fixtures.GetApplicationFixtures(testmigration.MorseUnbondingApplication)[0]
	unbondedAppFixture := fixtures.GetApplicationFixtures(testmigration.MorseUnbondedApplication)[0]

	s.Run("application unbonding began", func() {
		shannonDestAddr := sample.AccAddress()

		morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
			shannonDestAddr,
			unbondingAppFixture.GetPrivateKey(),
			s.appServiceConfig,
			sample.AccAddress(),
		)
		s.NoError(err)

		// Retrieve the unbonding application's onchain Morse claimable account.
		morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())

		// Calculate the expected unbonding session end height.
		estimatedBlockDuration, ok := pocket.EstimatedBlockDurationByChainId[s.GetApp().GetSdkCtx().ChainID()]
		require.Truef(s.T(), ok, "chain ID %s not found in EstimatedBlockDurationByChainId", s.GetApp().GetSdkCtx().ChainID())

		// Calculate the current session end height and the next session start height.
		currentHeight := s.GetApp().GetSdkCtx().BlockHeight()
		sharedParams := s.GetSharedParams(s.T())
		secondsUntilUnstakeCompletion := morseClaimableAccount.SecondsUntilUnbonded(s.SdkCtx())
		estimatedBlocksUntilUnstakeCompletion := secondsUntilUnstakeCompletion / int64(estimatedBlockDuration)
		estimatedUnstakeCompletionHeight := currentHeight + estimatedBlocksUntilUnstakeCompletion
		expectedUnstakeSessionEndHeight := uint64(sharedtypes.GetSessionEndHeight(&sharedParams, estimatedUnstakeCompletionHeight))

		// Calculate what the expect Supplier onchain should look like.
		expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight())
		expectedAppStake := morseClaimableAccount.GetApplicationStake()
		expectedApp := &apptypes.Application{
			Address:                   shannonDestAddr,
			Stake:                     &expectedAppStake,
			ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			UnstakeSessionEndHeight:   expectedUnstakeSessionEndHeight,
			DelegateeGatewayAddresses: make([]string, 0),
			PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
		}

		// Claim a Morse claimable account.
		morseClaimRes, err := s.GetApp().RunMsg(s.T(), morseClaimMsg)
		s.NoError(err)

		// Assert that the expected events were emitted.
		expectedMorseAppClaimEvent := &migrationtypes.EventMorseApplicationClaimed{
			MorseSrcAddress:         morseClaimMsg.GetMorseSignerAddress(),
			ClaimedBalance:          morseClaimableAccount.GetUnstakedBalance().String(),
			ClaimedApplicationStake: expectedAppStake.String(),
			SessionEndHeight:        expectedSessionEndHeight,
			ApplicationAddress:      expectedApp.Address,
		}

		expectedAppUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			ApplicationAddress: expectedApp.Address,
			Stake:              expectedApp.Stake.String(),
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   expectedSessionEndHeight,
			UnbondingEndHeight: int64(expectedUnstakeSessionEndHeight),
		}

		morseAppClaimedEvents := events.FilterEvents[*migrationtypes.EventMorseApplicationClaimed](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(morseAppClaimedEvents))
		require.Equal(s.T(), expectedMorseAppClaimEvent, morseAppClaimedEvents[0])

		appUnbondingBeginEvent := events.FilterEvents[*apptypes.EventApplicationUnbondingBegin](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(appUnbondingBeginEvent))
		require.Equal(s.T(), expectedAppUnbondingBeginEvent, appUnbondingBeginEvent[0])

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedApp.DelegateeGatewayAddresses = nil
		expectedApp.PendingUndelegations = nil

		// Check the Morse claim response.
		expectedMorseClaimRes := &migrationtypes.MsgClaimMorseApplicationResponse{}
		s.Equal(expectedMorseClaimRes, morseClaimRes)

		// Assert that the morseClaimableAccount is updated on-chain.
		expectedMorseClaimableAccount := morseClaimableAccount
		expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
		expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
		updatedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(expectedMorseClaimableAccount, updatedMorseClaimableAccount)

		// Assert that the application is unbonding.
		expectedApp = &apptypes.Application{
			Address:                 shannonDestAddr,
			Stake:                   &expectedAppStake,
			ServiceConfigs:          []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			UnstakeSessionEndHeight: expectedUnstakeSessionEndHeight,
		}
		appClient := s.AppSuite.GetAppQueryClient(s.T())
		foundApp, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)
		s.Equal(expectedApp, &foundApp)

		// Query for the application unstaked balance.
		bankClient := s.GetBankQueryClient(s.T())
		shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)
		s.Equal(morseClaimableAccount.GetUnstakedBalance(), *shannonDestBalance)
	})

	s.Run("application unbonding ended", func() {
		shannonDestAddr := sample.AccAddress()

		morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
			shannonDestAddr,
			unbondedAppFixture.GetPrivateKey(),
			s.appServiceConfig,
			sample.AccAddress(),
		)
		s.NoError(err)

		// Retrieve the unbonding application's onchain Morse claimable account.
		morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())

		// Calculate the expected unbonded session end height (previous session end).
		sharedParams := s.GetSharedParams(s.T())
		currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, s.GetApp().GetSdkCtx().BlockHeight())
		expectedUnstakeSessionEndHeight := uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStartHeight-1))

		expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight())
		expectedAppStake := morseClaimableAccount.GetApplicationStake()
		expectedApp := &apptypes.Application{
			Address:                   shannonDestAddr,
			Stake:                     &expectedAppStake,
			UnstakeSessionEndHeight:   expectedUnstakeSessionEndHeight,
			DelegateeGatewayAddresses: make([]string, 0),
			PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
			ServiceConfigs:            make([]*sharedtypes.ApplicationServiceConfig, 0),
		}

		// Claim a Morse claimable account.
		morseClaimRes, err := s.GetApp().RunMsg(s.T(), morseClaimMsg)
		s.NoError(err)

		// Assert that the expected events were emitted.
		expectedMorseAppClaimEvent := &migrationtypes.EventMorseApplicationClaimed{
			MorseSrcAddress:         morseClaimMsg.GetMorseSignerAddress(),
			ClaimedBalance:          morseClaimableAccount.GetUnstakedBalance().String(),
			ClaimedApplicationStake: expectedAppStake.String(),
			SessionEndHeight:        expectedSessionEndHeight,
			ApplicationAddress:      expectedApp.Address,
		}
		expectedAppUnbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
			ApplicationAddress: expectedApp.Address,
			Stake:              expectedApp.Stake.String(),
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   expectedSessionEndHeight,
			UnbondingEndHeight: int64(expectedUnstakeSessionEndHeight),
		}
		morseAppClaimedEvents := events.FilterEvents[*migrationtypes.EventMorseApplicationClaimed](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(morseAppClaimedEvents))
		require.Equal(s.T(), expectedMorseAppClaimEvent, morseAppClaimedEvents[0])
		appUnbondingEndEvent := events.FilterEvents[*apptypes.EventApplicationUnbondingEnd](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(appUnbondingEndEvent))
		require.Equal(s.T(), expectedAppUnbondingEndEvent, appUnbondingEndEvent[0])

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedApp.DelegateeGatewayAddresses = nil
		expectedApp.PendingUndelegations = nil
		expectedApp.ServiceConfigs = nil

		expectedMorseClaimRes := &migrationtypes.MsgClaimMorseApplicationResponse{}
		s.Equal(expectedMorseClaimRes, morseClaimRes)

		// Assert that the morseClaimableAccount is updated on-chain.
		expectedMorseClaimableAccount := morseClaimableAccount
		expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
		expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
		updatedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(expectedMorseClaimableAccount, updatedMorseClaimableAccount)

		// Assert that the application was unbonded (i.e.not staked).
		appClient := s.AppSuite.GetAppQueryClient(s.T())
		expectedErr := status.Error(
			codes.NotFound,
			apptypes.ErrAppNotFound.Wrapf(
				"app address: %s",
				shannonDestAddr,
			).Error(),
		)
		_, err = appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
		s.EqualError(err, expectedErr.Error())

		// Query for the application unstaked balance.
		bankClient := s.GetBankQueryClient(s.T())
		shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)
		s.Equal(morseClaimableAccount.TotalTokens(), *shannonDestBalance)
	})
}
