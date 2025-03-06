package migration

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type MigrationModuleTestSuite struct {
	suites.MigrationModuleSuite

	// numMorseClaimableAccounts is the number of morse claimable accounts to
	// generate when calling #GenerateMorseAccountState.
	numMorseClaimableAccounts int
}

func (s *MigrationModuleTestSuite) SetupTest() {
	// Initialize a new integration app for the suite.
	s.NewApp(s.T())

	s.numMorseClaimableAccounts = 10

	// Assign the app to nested suites.
	// TODO_UPNEXT(@bryanchriswhite, #1043): Initialize the app module suite.
	// s.AppSuite.SetApp(s.GetApp())
}

func TestMigrationModuleSuite(t *testing.T) {
	suite.Run(t, &MigrationModuleTestSuite{})
}

// TestImportMorseClaimableAccounts exercises importing and persistence of morse claimable accounts.
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)
	msgImportRes := s.ImportMorseClaimableAccounts(s.T())
	morseAccountState := s.GetAccountState(s.T())
	morseAccountStateHash, err := morseAccountState.GetHash()
	require.NoError(s.T(), err)

	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   morseAccountStateHash,
		NumAccounts: uint64(s.numMorseClaimableAccounts),
	}
	require.Equal(s.T(), expectedMsgImportRes, msgImportRes)

	foundMorseClaimableAccounts := s.QueryAllMorseClaimableAccounts(s.T())
	require.Equal(s.T(), s.numMorseClaimableAccounts, len(foundMorseClaimableAccounts))

	for _, expectedMorseClaimableAccount := range morseAccountState.Accounts {
		isFound := false
		for _, foundMorseClaimableAccount := range foundMorseClaimableAccounts {
			if foundMorseClaimableAccount.GetMorseSrcAddress() == expectedMorseClaimableAccount.GetMorseSrcAddress() {
				require.Equal(s.T(), *expectedMorseClaimableAccount, foundMorseClaimableAccount)
				isFound = true
				break
			}
		}
		require.True(s.T(), isFound)
	}
}

// TestClaimMorseAccount exercises claiming of MorseClaimableAccounts.
// It only exercises claiming of account balances and does not exercise
// the staking any actors as a result of claiming.
func (s *MigrationModuleTestSuite) TestClaimMorseAccount() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)
	s.ImportMorseClaimableAccounts(s.T())

	shannonDestAddr := sample.AccAddress()

	bankClient := s.GetBankQueryClient(s.T())
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.True(s.T(), shannonDestBalance.IsZero())

	morseSrcAddr, claimAccountRes := s.ClaimMorseAccount(s.T(), 0, shannonDestAddr)

	expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[0]
	expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
		Add(expectedMorseClaimableAccount.GetApplicationStake()).
		Add(expectedMorseClaimableAccount.GetSupplierStake())

	s.GetSharedParams(s.T())
	sharedParams := s.GetSharedParams(s.T())
	currentHeight := s.SdkCtx().BlockHeight()
	expectedSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	expectedClaimAccountRes := &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress:  morseSrcAddr,
		ClaimedBalance:   expectedBalance,
		SessionEndHeight: expectedSessionEndHeight,
	}
	require.Equal(s.T(), expectedClaimAccountRes, claimAccountRes)

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
	require.Equal(s.T(), sdk.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)
}
