package suites

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ IntegrationSuite = (*MigrationModuleSuite)(nil)

// MigrationModuleSuite is a test suite which abstracts common migration module
// functionality. It is intended to be embedded in dependent integration test suites.
type MigrationModuleSuite struct {
	BaseIntegrationSuite
	AppSuite      ApplicationModuleSuite
	SupplierSuite SupplierModuleSuite
	ServiceSuite  ServiceModuleSuite

	// accountState is the generated MorseAccountState to be imported into the migration module.
	accountState *migrationtypes.MorseAccountState

	// numMorseClaimableAccounts is the number of morse claimable accounts to generate when calling #GenerateMorseAccountState.
	numMorseClaimableAccounts int
}

// GenerateMorseAccountState generates a MorseAccountState with the given number of MorseClaimableAccounts.
// It updates the suite's #numMorseClaimableAccounts and #accountState fields.
func (s *MigrationModuleSuite) GenerateMorseAccountState(t *testing.T, numAccounts int, distributionFn testmigration.MorseAccountActorTypeDistributionFn) {
	s.numMorseClaimableAccounts = numAccounts
	var err error
	_, s.accountState, err = testmigration.NewMorseStateExportAndAccountState(s.numMorseClaimableAccounts, distributionFn)
	require.NoError(t, err)
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
func (s *MigrationModuleSuite) ClaimMorseAccount(
	t *testing.T,
	morseAccountIdx uint64,
	shannonDestAddr string,
	signerAddr string,
) (expectedMorseSrcAddr string, _ *migrationtypes.MsgClaimMorseAccountResponse) {
	t.Helper()

	morsePrivateKey := testmigration.GenMorsePrivateKey(morseAccountIdx)
	expectedMorseSrcAddr = morsePrivateKey.PubKey().Address().String()
	require.Equal(t, expectedMorseSrcAddr, s.accountState.Accounts[morseAccountIdx].MorseSrcAddress)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseAccount(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		signerAddr,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account.
	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	claimAccountRes, ok := resAny.(*migrationtypes.MsgClaimMorseAccountResponse)
	require.True(t, ok)

	return expectedMorseSrcAddr, claimAccountRes
}

// MorseClaimableAccountQuerier returns a migration module query client for morse claimable accounts.
func (s *MigrationModuleSuite) MorseClaimableAccountQuerier() migrationtypes.QueryClient {
	return migrationtypes.NewQueryClient(s.GetApp().QueryHelper())
}

// QueryMorseClaimableAccount queries the migration module for the given morseSrcAddr.
func (s *MigrationModuleSuite) QueryMorseClaimableAccount(
	t *testing.T,
	morseSrcAddr string,
) *migrationtypes.MorseClaimableAccount {
	t.Helper()

	morseAccountQuerier := s.MorseClaimableAccountQuerier()
	morseClaimableAcctRes, err := morseAccountQuerier.MorseClaimableAccount(
		s.SdkCtx(),
		&migrationtypes.QueryMorseClaimableAccountRequest{
			Address: morseSrcAddr,
		},
	)
	require.NoError(t, err)

	return &morseClaimableAcctRes.MorseClaimableAccount
}

// QueryAllMorseClaimableAccounts queries the migration module for all morse claimable accounts.
func (s *MigrationModuleSuite) QueryAllMorseClaimableAccounts(t *testing.T) []migrationtypes.MorseClaimableAccount {
	t.Helper()

	morseAccountQuerier := s.MorseClaimableAccountQuerier()
	morseClaimableAcctRes, err := morseAccountQuerier.MorseClaimableAccountAll(
		s.SdkCtx(),
		&migrationtypes.QueryAllMorseClaimableAccountRequest{},
	)
	require.NoError(t, err)

	return morseClaimableAcctRes.MorseClaimableAccount
}

// GetSharedParams returns the shared module params.
func (s *MigrationModuleSuite) GetSharedParams(t *testing.T) sharedtypes.Params {
	sharedClient := sharedtypes.NewQueryClient(s.GetApp().QueryHelper())
	sharedParamsRes, err := sharedClient.Params(s.SdkCtx(), &sharedtypes.QueryParamsRequest{})
	require.NoError(t, err)

	return sharedParamsRes.Params
}

// GetSessionEndHeight returns the session end height for the given query height.
func (s *MigrationModuleSuite) GetSessionEndHeight(t *testing.T, queryHeight int64) int64 {
	sharedParams := s.GetSharedParams(t)
	return sharedtypes.GetSessionEndHeight(&sharedParams, queryHeight)
}

// ClaimMorseApplication claims the given MorseClaimableAccount as a staked application
// by running a MsgClaimMorseApplication message.
// It returns the expected Morse source address and the MsgClaimMorseApplicationResponse.
func (s *MigrationModuleSuite) ClaimMorseApplication(
	t *testing.T,
	morseAccountIdx uint64,
	shannonDestAddr string,
	serviceConfig *sharedtypes.ApplicationServiceConfig,
	signingAddr string,
) (expectedMorseSrcAddr string, _ *migrationtypes.MsgClaimMorseApplicationResponse) {
	t.Helper()

	morsePrivateKey := testmigration.GenMorsePrivateKey(morseAccountIdx)
	expectedMorseSrcAddr = morsePrivateKey.PubKey().Address().String()
	require.Equal(t,
		expectedMorseSrcAddr,
		s.accountState.Accounts[morseAccountIdx].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseApplication(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		serviceConfig,
		signingAddr,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account as an application.
	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	claimApplicationRes, ok := resAny.(*migrationtypes.MsgClaimMorseApplicationResponse)
	require.True(t, ok)

	return expectedMorseSrcAddr, claimApplicationRes
}

// ClaimMorseSupplier claims the given MorseClaimableAccount as a staked supplier
// by running a MsgClaimMorseSupplier message.
// It returns the expected Morse source address and the MsgClaimMorseSupplierResponse.
func (s *MigrationModuleSuite) ClaimMorseSupplier(
	t *testing.T,
	morseAccountIdx uint64,
	shannonDestAddr string,
	services []*sharedtypes.SupplierServiceConfig,
	signingAddr string,
) (expectedMorseSrcAddr string, _ *migrationtypes.MsgClaimMorseSupplierResponse) {
	t.Helper()

	morsePrivateKey := testmigration.GenMorsePrivateKey(morseAccountIdx)
	expectedMorseSrcAddr = morsePrivateKey.PubKey().Address().String()
	require.Equal(t,
		expectedMorseSrcAddr,
		s.accountState.Accounts[morseAccountIdx].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
		shannonDestAddr,
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		services,
		signingAddr,
	)
	require.NoError(t, err)

	// Claim a Morse claimable account as a supplier.
	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	claimSupplierRes, ok := resAny.(*migrationtypes.MsgClaimMorseSupplierResponse)
	require.True(t, ok)

	return expectedMorseSrcAddr, claimSupplierRes
}
