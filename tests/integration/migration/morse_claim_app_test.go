package migration

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
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

var (
	stakeOffset = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 9999)

	testCases = []struct {
		desc     string
		getStake func(s *MigrationModuleTestSuite) *cosmostypes.Coin
	}{
		{
			desc:     "claim morse application with same staked/unstaked ratio (default)",
			getStake: func(_ *MigrationModuleTestSuite) *cosmostypes.Coin { return nil },
		},
		{
			desc: "claim morse application with same staked/unstaked ratio (explicit)",
			getStake: func(s *MigrationModuleTestSuite) *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[1].
					GetApplicationStake()
				return &stake
			},
		},
		{
			desc: "claim morse application with higher staked/unstaked ratio",
			getStake: func(s *MigrationModuleTestSuite) *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[2].
					GetApplicationStake().
					Add(stakeOffset)
				return &stake
			},
		},
		{
			desc: "claim morse application with lower staked/unstaked ratio",
			getStake: func(s *MigrationModuleTestSuite) *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[3].
					GetApplicationStake().
					Sub(stakeOffset)
				return &stake
			},
		},
	}
)

// TestClaimMorseApplication exercises claiming of a MorseClaimableAccount as a new staked application.
func (s *MigrationModuleTestSuite) TestClaimMorseNewApplication() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	for testCaseIdx, testCase := range testCases {
		s.T().Run(testCase.desc, func(t *testing.T) {
			shannonDestAddr := sample.AccAddress()
			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as a new application.
			morseSrcAddr, claimAppRes := s.ClaimMorseApplication(
				s.T(), uint64(testCaseIdx+1),
				shannonDestAddr,
				testCase.getStake(s),
				s.appServiceConfig,
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[testCaseIdx]
			expectedStake := testCase.getStake(s)
			if expectedStake == nil {
				expectedStake = &expectedMorseClaimableAccount.ApplicationStake
			}

			// Assert that the claim msg response is correct.
			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
				Add(expectedMorseClaimableAccount.GetApplicationStake()).
				Add(expectedMorseClaimableAccount.GetSupplierStake()).
				Sub(*expectedStake)

			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{
				MorseSrcAddress:         morseSrcAddr,
				ClaimedBalance:          expectedBalance,
				ClaimedApplicationStake: *expectedStake,
				ClaimedAtHeight:         s.SdkCtx().BlockHeight() - 1,
				ServiceId:               s.appServiceConfig.GetServiceId(),
			}
			require.Equal(s.T(), expectedClaimApplicationRes, claimAppRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			require.Equal(s.T(), expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			require.NoError(s.T(), err)
			require.Equal(s.T(), cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the application was staked.
			expectedApp := apptypes.Application{
				Address:        shannonDestAddr,
				Stake:          expectedStake,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			}
			appClient := s.AppSuite.GetAppQueryClient(s.T())
			app, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), expectedApp, app)
		})
	}
}

// TestClaimMorseApplication exercises claiming of a MorseClaimableAccount as an existing staked application.
func (s *MigrationModuleTestSuite) TestClaimMorseExistingApplication() {
	// Generate and import Morse claimable accounts.
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	for testCaseIdx, testCase := range testCases {
		s.T().Run(testCase.desc, func(t *testing.T) {
			// Stake an initial application.
			shannonDestAddr := sample.AccAddress()
			shannonDestAccAddr := cosmostypes.MustAccAddressFromBech32(shannonDestAddr)
			appClient := s.AppSuite.GetAppQueryClient(s.T())
			appParams, err := appClient.GetParams(s.SdkCtx())
			require.NoError(s.T(), err)

			initialAppStake := appParams.GetMinStake()
			s.FundAddress(s.T(), shannonDestAccAddr, initialAppStake.Amount.Int64())
			s.AppSuite.StakeApp(s.T(), shannonDestAddr, initialAppStake.Amount.Int64(), []string{"nosvc"})

			// Assert that the initial application is staked.
			foundApp, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), shannonDestAddr, foundApp.Address)

			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as an existing application.
			morseSrcAddr, claimAppRes := s.ClaimMorseApplication(
				s.T(), uint64(testCaseIdx+1),
				shannonDestAddr,
				testCase.getStake(s),
				s.appServiceConfig,
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[testCaseIdx]
			appStakeToClaim := testCase.getStake(s)
			if appStakeToClaim == nil {
				appStakeToClaim = &expectedMorseClaimableAccount.ApplicationStake
			}

			expectedClaimedStake := appStakeToClaim.Sub(*initialAppStake)
			expectedBalance := expectedMorseClaimableAccount.TotalTokens().
				Sub(expectedClaimedStake)

			// Assert that the claim msg response is correct.
			expectedClaimApplicationRes := &migrationtypes.MsgClaimMorseApplicationResponse{
				MorseSrcAddress:         morseSrcAddr,
				ClaimedBalance:          expectedBalance,
				ClaimedApplicationStake: expectedClaimedStake,
				ClaimedAtHeight:         s.SdkCtx().BlockHeight() - 1,
				ServiceId:               s.appServiceConfig.GetServiceId(),
			}
			require.Equal(s.T(), expectedClaimApplicationRes, claimAppRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			require.Equal(s.T(), expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			require.NoError(s.T(), err)
			require.Equal(s.T(), cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the application was updated.
			expectedApp := apptypes.Application{
				Address:        shannonDestAddr,
				Stake:          appStakeToClaim,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{s.appServiceConfig},
			}
			app, err := appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), expectedApp, app)
		})
	}
}

func (s *MigrationModuleTestSuite) TestClaimMorseApplication_ErrorMinStake() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	appClient := s.AppSuite.GetAppQueryClient(s.T())
	appParams, err := appClient.GetParams(s.SdkCtx())
	appMinStake := appParams.GetMinStake()
	require.NoError(s.T(), err)
	belowAppMinStake := appMinStake.Sub(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.NewMorsePrivateKey(s.T(), 1)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		&belowAppMinStake,
		s.appServiceConfig,
	)
	require.NoError(s.T(), err)

	// Claim a Morse claimable account.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	require.Contains(s.T(), strings.ReplaceAll(err.Error(), `\`, ""), status.Error(
		codes.InvalidArgument,
		apptypes.ErrAppInvalidStake.Wrapf("application %q must stake at least %s",
			shannonDestAddr, appMinStake,
		).Error(),
	).Error())

	// Assert that the MorseClaimableAccount was NOT updated on-chain.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSrcAddress())
	require.Equal(s.T(), int64(0), morseClaimableAccount.GetClaimedAtHeight())
	require.Equal(s.T(), "", morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance has NOT been updated.
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	require.NoError(s.T(), err)
	require.Equal(s.T(), cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the application was NOT staked.
	_, err = appClient.GetApplication(s.SdkCtx(), shannonDestAddr)
	require.EqualError(s.T(), err, status.Error(
		codes.NotFound,
		apptypes.ErrAppNotFound.Wrapf(
			"app address: %s",
			shannonDestAddr,
		).Error(),
	).Error())
}

func (s *MigrationModuleTestSuite) TestClaimMorseApplication_ErrorInsufficientStakeAvailable() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	//aboveMaxAvailableStake := appMinStake.Sub(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))
	expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[0]
	totalTokens := expectedMorseClaimableAccount.TotalTokens()
	aboveMaxAvailableStake := totalTokens.Add(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.NewMorsePrivateKey(s.T(), 1)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		&aboveMaxAvailableStake,
		s.appServiceConfig,
	)
	require.NoError(s.T(), err)

	// Claim a Morse claimable account.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	require.ErrorContains(s.T(), err, status.Error(
		codes.Internal,
		errors.ErrInsufficientFunds.Wrapf("spendable balance %s is smaller than %s",
			totalTokens, aboveMaxAvailableStake,
		).Error(),
	).Error())

	// Assert that the MorseClaimableAccount was NOT updated on-chain.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSrcAddress())
	require.Equal(s.T(), int64(0), morseClaimableAccount.GetClaimedAtHeight())
	require.Equal(s.T(), "", morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance has NOT been updated.
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	require.NoError(s.T(), err)
	require.Equal(s.T(), cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

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
