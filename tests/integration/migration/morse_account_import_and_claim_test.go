package migration

import (
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func TestMsgServer_CreateMorseAccountClaim(t *testing.T) {
	app := integration.NewCompleteIntegrationApp(t)

	// Generate Morse claimable accounts.
	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)

	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*accountState,
	)
	require.NoError(t, err)

	// Import Morse claimable accounts.
	resAny, err := app.RunMsg(t, msgImport)
	require.NoError(t, err)

	msgImportRes, ok := resAny.(*migrationtypes.MsgImportMorseClaimableAccountsResponse)
	require.True(t, ok)

	morseAccountStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   morseAccountStateHash,
		NumAccounts: uint64(numAccounts),
	}
	require.Equal(t, expectedMsgImportRes, msgImportRes)

	deps := depinject.Supply(app.QueryHelper())
	bankClient, err := query.NewBankQuerier(deps)
	require.NoError(t, err)

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestAddr := sample.AccAddress()
	shannonDestBalance, err := bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
	require.NoError(t, err)
	require.Equal(t, int64(0), shannonDestBalance.Amount.Int64())

	morsePrivateKey := testmigration.NewMorsePrivateKey(t, 1)
	morseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(t, morseSrcAddr, accountState.Accounts[0].MorseSrcAddress)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseAccount(
		shannonDestAddr,
		morseSrcAddr,
		morsePrivateKey,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account.
	resAny, err = app.RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	expectedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1110111)
	expectedClaimAccountRes := &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress: morseSrcAddr,
		ClaimedBalance:  expectedBalance,
		ClaimedAtHeight: app.GetSdkCtx().BlockHeight() - 1,
	}

	claimAccountRes, ok := resAny.(*migrationtypes.MsgClaimMorseAccountResponse)
	require.True(t, ok)
	require.Equal(t, expectedClaimAccountRes, claimAccountRes)

	// Assert that the MorseClaimableAccount was updated on-chain.
	expectedMorseClaimableAccount := *accountState.Accounts[0]
	expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
	expectedMorseClaimableAccount.ClaimedAtHeight = app.GetSdkCtx().BlockHeight() - 1

	morseAccountQuerier := migrationtypes.NewQueryClient(app.QueryHelper())
	morseClaimableAcctRes, err := morseAccountQuerier.MorseClaimableAccount(app.GetSdkCtx(), &migrationtypes.QueryMorseClaimableAccountRequest{
		Address: morseSrcAddr,
	})
	require.NoError(t, err)
	require.Equal(t, expectedMorseClaimableAccount, morseClaimableAcctRes.MorseClaimableAccount)

	// Assert that the shannonDestAddr account balance has been updated.
	shannonDestBalance, err = bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
	require.NoError(t, err)
	require.Equal(t, expectedBalance, *shannonDestBalance)

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(app.GetSdkCtx(), migrationModuleAddress)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)
}
