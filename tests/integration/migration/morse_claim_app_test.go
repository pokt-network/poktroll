package migration

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestClaimMorseApplication exercises claiming of a MorseClaimableAccount as a staked application.
func (s *MigrationModuleTestSuite) TestClaimMorseApplication() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	stakeOffset := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 9999)
	testCases := []struct {
		desc     string
		getStake func() *cosmostypes.Coin
	}{
		{
			desc:     "claim morse application with same staked/unstaked ratio (default)",
			getStake: func() *cosmostypes.Coin { return nil },
		},
		{
			desc: "claim morse application with same staked/unstaked ratio (explicit)",
			getStake: func() *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[1].
					GetApplicationStake()
				return &stake
			},
		},
		{
			desc: "claim morse application with higher staked/unstaked ratio",
			getStake: func() *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[2].
					GetApplicationStake().
					Add(stakeOffset)
				return &stake
			},
		},
		{
			desc: "claim morse application with lower staked/unstaked ratio",
			getStake: func() *cosmostypes.Coin {
				stake := s.GetAccountState(s.T()).Accounts[3].
					GetApplicationStake().
					Sub(stakeOffset)
				return &stake
			},
		},
	}

	for testCaseIdx, testCase := range testCases {
		s.T().Run(testCase.desc, func(t *testing.T) {
			shannonDestAddr := sample.AccAddress()
			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			require.NoError(s.T(), err)
			require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

			morseSrcAddr, claimAppRes := s.ClaimMorseApplication(
				s.T(), uint64(testCaseIdx+1),
				shannonDestAddr,
				testCase.getStake(),
				s.appServiceConfig,
			)

			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[testCaseIdx]
			expectedStake := testCase.getStake()
			if expectedStake == nil {
				expectedStake = &expectedMorseClaimableAccount.ApplicationStake
			}

			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
				Add(expectedMorseClaimableAccount.GetApplicationStake()).
				Add(expectedMorseClaimableAccount.GetSupplierStake()).
				Sub(*expectedStake)

			expectedClaimARes := &migrationtypes.MsgClaimMorseApplicationResponse{
				MorseSrcAddress:         morseSrcAddr,
				ClaimedBalance:          expectedBalance,
				ClaimedApplicationStake: *expectedStake,
				ClaimedAtHeight:         s.SdkCtx().BlockHeight() - 1,
				ServiceId:               s.appServiceConfig.GetServiceId(),
			}
			require.Equal(s.T(), expectedClaimARes, claimAppRes)

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

// TODO_IN_THIS_COMMIT: error cases...
// - stake is below min stake
// - stake is greater than available total tokens

//func TestMsgServer_CreateMorseApplicationClaim(t *testing.T) {
//	expectedAppServiceConfig := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}
//	expectedAppStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)
//
//	app := integration.NewCompleteIntegrationApp(t)
//
//	// Generate Morse claimable accounts.
//	numAccounts := 10
//	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)
//
//	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
//		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
//		*accountState,
//	)
//	require.NoError(t, err)
//
//	// Import Morse claimable accounts.
//	resAny, err := app.RunMsg(t, msgImport)
//	require.NoError(t, err)
//
//	msgImportRes, ok := resAny.(*migrationtypes.MsgImportMorseClaimableAccountsResponse)
//	require.True(t, ok)
//
//	morseAccountStateHash, err := accountState.GetHash()
//	require.NoError(t, err)
//
//	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
//		StateHash:   morseAccountStateHash,
//		NumAccounts: uint64(numAccounts),
//	}
//	require.Equal(t, expectedMsgImportRes, msgImportRes)
//
//	deps := depinject.Supply(app.QueryHelper())
//	bankClient, err := query.NewBankQuerier(deps)
//	require.NoError(t, err)
//
//	// Assert that the shannonDestAddr account initially has a zero balance.
//	shannonDestAddr := sample.AccAddress()
//	shannonDestBalance, err := bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
//	require.NoError(t, err)
//	require.Equal(t, int64(0), shannonDestBalance.Amount.Int64())
//
//	morsePrivateKey := testmigration.NewMorsePrivateKey(t, 1)
//	morseSrcAddr := morsePrivateKey.PubKey().Address().String()
//	require.Equal(t, morseSrcAddr, accountState.Accounts[0].MorseSrcAddress)
//
//	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
//		shannonDestAddr,
//		morseSrcAddr,
//		morsePrivateKey,
//		expectedAppStake,
//		expectedAppServiceConfig,
//	)
//	require.NoError(t, err)
//
//	// Assert that an application with the ShannonDestAddress does not initially exist.
//	appQuerier, err := query.NewApplicationQuerier(deps)
//	require.NoError(t, err)
//
//	_, err = appQuerier.GetApplication(app.GetSdkCtx(), shannonDestAddr)
//	require.EqualError(t, err, status.Error(
//		codes.NotFound,
//		types.ErrAppNotFound.Wrapf("app with address %q not found", shannonDestAddr).Error(),
//	).Error())
//
//	// Claim a Morse claimable account as a staked application.
//	resAny, err = app.RunMsg(t, morseClaimMsg)
//	require.NoError(t, err)
//
//	expectedBalance := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1010101)
//	expectedClaimAccountRes := &migrationtypes.MsgClaimMorseApplicationResponse{
//		MorseSrcAddress:         morseSrcAddr,
//		ClaimedBalance:          expectedBalance,
//		ClaimedApplicationStake: expectedAppStake,
//		ClaimedAtHeight:         app.GetSdkCtx().BlockHeight() - 1,
//		ServiceId:               expectedAppServiceConfig.GetServiceId(),
//	}
//
//	claimAccountRes, ok := resAny.(*migrationtypes.MsgClaimMorseApplicationResponse)
//	assert.True(t, ok)
//	require.Equal(t, expectedClaimAccountRes, claimAccountRes)
//
//	// Assert that the MorseClaimableAccount was updated on-chain.
//	expectedMorseClaimableAccount := *accountState.Accounts[0]
//	expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
//	expectedMorseClaimableAccount.ClaimedAtHeight = app.GetSdkCtx().BlockHeight() - 1
//
//	morseAccountQuerier := migrationtypes.NewQueryClient(app.QueryHelper())
//	morseClaimableAcctRes, err := morseAccountQuerier.MorseClaimableAccount(app.GetSdkCtx(), &migrationtypes.QueryMorseClaimableAccountRequest{
//		Address: morseSrcAddr,
//	})
//	require.NoError(t, err)
//	require.Equal(t, expectedMorseClaimableAccount, morseClaimableAcctRes.MorseClaimableAccount)
//
//	// Assert that the shannonDestAddr account balance has been updated.
//	shannonDestBalance, err = bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
//	require.NoError(t, err)
//	require.Equal(t, expectedBalance, *shannonDestBalance)
//
//	// Assert that the migration module account balance returns to zero.
//	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
//	migrationModuleBalance, err := bankClient.GetBalance(app.GetSdkCtx(), migrationModuleAddress)
//	require.NoError(t, err)
//	require.Equal(t, cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)
//}
