package migration

import (
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/sample"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// TestClaimMorseAccount exercises claiming of MorseClaimableAccounts.
// It only exercises claiming of account balances and does not exercise
// the staking any actors as a result of claiming.
func (s *MigrationModuleTestSuite) TestClaimMorseAccount() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	require.NoError(s.T(), err)
	require.True(s.T(), shannonDestBalance.IsZero())

	morseSrcAddr, claimAccountRes := s.ClaimMorseAccount(s.T(), 0, shannonDestAddr)

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
	require.Equal(s.T(), cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)
}
