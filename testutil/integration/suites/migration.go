package suites

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

var _ IntegrationSuite = (*MigrationModuleSuite)(nil)

// MigrationModuleSuite is a test suite which abstracts common migration module
// functionality. It is intended to be embedded in dependent integration test suites.
type MigrationModuleSuite struct {
	BaseIntegrationSuite
	// TODO_UPNEXT(@bryanchriswhite, #1043): Add ApplicationModuleSuite to the suite.
	// AppSuite ApplicationModuleSuite

	// accountState is the generated MorseAccountState to be imported into the migration module.
	accountState *migrationtypes.MorseAccountState
	// numMorseClaimableAccounts is the number of morse claimable accounts to generate when calling #GenerateMorseAccountState.
	numMorseClaimableAccounts int
}

// GenerateMorseAccountState generates a MorseAccountState with the given number of MorseClaimableAccounts.
// It updates the suite's #numMorseClaimableAccounts and #accountState fields.
func (s *MigrationModuleSuite) GenerateMorseAccountState(t *testing.T, numAccounts int) {
	s.numMorseClaimableAccounts = numAccounts
	_, s.accountState = testmigration.NewMorseStateExportAndAccountState(t, s.numMorseClaimableAccounts)
}

// GetAccountState returns the suite's #accountState field.
func (s *MigrationModuleSuite) GetAccountState(t *testing.T) *migrationtypes.MorseAccountState {
	require.NotNil(t, s.accountState)
	return s.accountState
}

// ImportMorseClaimableAccounts imports the MorseClaimableAccounts from the suite's
// #accountState field by running a MsgImportMorseClaimableAccounts message.
func (s *MigrationModuleSuite) ImportMorseClaimableAccounts(t *testing.T) *migrationtypes.MsgImportMorseClaimableAccountsResponse {
	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*s.accountState,
	)
	require.NoError(t, err)

	// Import Morse claimable accounts.
	resAny, err := s.GetApp().RunMsg(t, msgImport)
	require.NoError(t, err)

	msgImportRes, ok := resAny.(*migrationtypes.MsgImportMorseClaimableAccountsResponse)
	require.True(t, ok)

	return msgImportRes
}

// ClaimMorseAccount claims the given MorseClaimableAccount by running a MsgClaimMorseAccount message.
// It returns the expected Morse source address and the MsgClaimMorseAccountResponse.
// DEV_NOTE: morseAccountIdx is 1-based.
func (s *MigrationModuleSuite) ClaimMorseAccount(
	t *testing.T,
	morseAccountIdx uint64,
	shannonDestAddr string,
) (expectedMorseSrcAddr string, _ *migrationtypes.MsgClaimMorseAccountResponse) {
	t.Helper()

	morsePrivateKey := testmigration.NewMorsePrivateKey(t, morseAccountIdx)
	expectedMorseSrcAddr = morsePrivateKey.PubKey().Address().String()
	require.Equal(t, expectedMorseSrcAddr, s.accountState.Accounts[0].MorseSrcAddress)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseAccount(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account.
	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	claimAccountRes, ok := resAny.(*migrationtypes.MsgClaimMorseAccountResponse)
	require.True(t, ok)

	return expectedMorseSrcAddr, claimAccountRes
}

// QueryMorseClaimableAccount queries the migration module for the given morseSrcAddr.
func (s *MigrationModuleSuite) QueryMorseClaimableAccount(
	t *testing.T,
	morseSrcAddr string,
) *migrationtypes.MorseClaimableAccount {
	t.Helper()

	morseAccountQuerier := migrationtypes.NewQueryClient(s.GetApp().QueryHelper())
	morseClaimableAcctRes, err := morseAccountQuerier.MorseClaimableAccount(
		s.SdkCtx(),
		&migrationtypes.QueryMorseClaimableAccountRequest{
			Address: morseSrcAddr,
		},
	)
	require.NoError(t, err)

	return &morseClaimableAcctRes.MorseClaimableAccount
}
