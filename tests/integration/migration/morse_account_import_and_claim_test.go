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
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

type MigrationModuleTestSuite struct {
	suites.MigrationModuleSuite

	// TODO_IN_THIS_COMMIT: godoc...
	numAccounts int
}

func (s *MigrationModuleTestSuite) SetupTest() {
	// Initialize a new integration app for the suite.
	s.NewApp(s.T())

	s.numAccounts = 10

	// Assign the app to nested suites.
	s.AppSuite.SetApp(s.GetApp())
}

func TestMigrationModuleSuite(t *testing.T) {
	suite.Run(t, &MigrationModuleTestSuite{})
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts() {
	s.GenerateMorseAccountState(s.T(), s.numAccounts)
	msgImportRes := s.ImportMorseClaimableAccounts(s.T())
	morseAccountStateHash, err := s.GetAccountState(s.T()).GetHash()
	require.NoError(s.T(), err)

	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   morseAccountStateHash,
		NumAccounts: uint64(s.numAccounts),
	}
	require.Equal(s.T(), expectedMsgImportRes, msgImportRes)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *MigrationModuleTestSuite) TestClaimMorseAccount() {
	s.GenerateMorseAccountState(s.T(), s.numAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	shannonDestAddr := sample.AccAddress()

	bankClient := s.GetBankQueryClient(s.T())
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), shannonDestBalance.Amount.Int64())

	morseSrcAddr, claimAccountRes := s.ClaimMorseAccount(s.T(), 1, shannonDestAddr)

	expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[0]
	expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
		Add(expectedMorseClaimableAccount.GetApplicationStake()).
		Add(expectedMorseClaimableAccount.GetSupplierStake())

	expectedClaimAccountRes := &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress: morseSrcAddr,
		ClaimedBalance:  expectedBalance,
		ClaimedAtHeight: s.SdkCtx().BlockHeight() - 1,
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
