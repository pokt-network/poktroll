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

// TODO_IN_THIS_COMMIT: godoc...
type MigrationModuleSuite struct {
	BaseIntegrationSuite
	// TODO_IN_THIS_COMMIT: godoc... set in #GenerateMorseAccountState(), used in #ImportMorseClaimableAccounts().
	accountState *migrationtypes.MorseAccountState
	// TODO_IN_THIS_COMMIT: godoc... set in #GenerateMorseAccountState(), used in #ImportMorseClaimableAccounts().
	numAccounts int

	// TODO_UPNEXT(@bryanchriswhite, #1043): Add ApplicationModuleSuite to the suite.
	// AppSuite ApplicationModuleSuite
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *MigrationModuleSuite) GenerateMorseAccountState(t *testing.T, numAccounts int) {
	s.numAccounts = numAccounts
	_, s.accountState = testmigration.NewMorseStateExportAndAccountState(t, s.numAccounts)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *MigrationModuleSuite) GetAccountState(t *testing.T) *migrationtypes.MorseAccountState {
	require.NotNil(t, s.accountState)
	return s.accountState
}

// TODO_IN_THIS_COMMIT: godoc...
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

// TODO_IN_THIS_COMMIT: godoc... NOTE: morseAccountIdx is 1-based...
func (s *MigrationModuleSuite) ClaimMorseAccount(
	t *testing.T,
	morseAccountIdx uint64,
	shannonDestAddr string,
) (morseSrcAddr string, _ *migrationtypes.MsgClaimMorseAccountResponse) {
	t.Helper()

	morsePrivateKey := testmigration.NewMorsePrivateKey(t, morseAccountIdx)
	morseSrcAddr = morsePrivateKey.PubKey().Address().String()
	require.Equal(t, morseSrcAddr, s.accountState.Accounts[0].MorseSrcAddress)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseAccount(
		shannonDestAddr,
		morseSrcAddr,
		morsePrivateKey,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account.
	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	claimAccountRes, ok := resAny.(*migrationtypes.MsgClaimMorseAccountResponse)
	require.True(t, ok)

	return morseSrcAddr, claimAccountRes
}

// TODO_IN_THIS_COMMIT: godoc...
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
