package migration

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
			expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight()-1)
			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{
				MorseSrcAddress:         morseSrcAddr,
				ClaimedBalance:          expectedBalance,
				ClaimedApplicationStake: expectedStake,
				SessionEndHeight:        expectedSessionEndHeight,
				Application:             &expectedApp,
			}
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
			s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

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

	for morseAccountIdx, _ := range s.GetAccountState(s.T()).Accounts {
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
			expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight()-1)
			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{
				MorseSrcAddress:         morseSrcAddr,
				ClaimedBalance:          expectedBalance,
				ClaimedApplicationStake: expectedClaimedStake,
				SessionEndHeight:        expectedSessionEndHeight,
				Application:             &expectedApp,
			}
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
			s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the application was updated.
			app, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedApp, app)
		})
	}
}

func (s *MigrationModuleTestSuite) TestClaimMorseApplication_ErrorMinStake() {
	// Set the min app stake param to just above the application stake amount.
	minStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, testmigration.GenMorseApplicationStakeAmount(uint64(0))+1)
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
	require.Error(s.T(), err)
	require.Contains(s.T(), strings.ReplaceAll(err.Error(), `\`, ""), status.Error(
		codes.InvalidArgument,
		apptypes.ErrAppInvalidStake.Wrapf("application %q must stake at least %s",
			shannonDestAddr, s.minStake,
		).Error(),
	).Error())

	// Assert that the MorseClaimableAccount was NOT updated on-chain.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSrcAddress())
	s.Equal(int64(0), morseClaimableAccount.GetClaimedAtHeight())
	s.Equal("", morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance has NOT been updated.
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	s.NoError(err)
	s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

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
