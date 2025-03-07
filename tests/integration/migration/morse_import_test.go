package migration

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	testServiceId                    = "svc1"
	mockMorseAccountStateHash        = "d7469245aabadc98330f79eef9fb544aa3df0c7cbeabfc3f994fd419b2661633"
	defaultNumMorseClaimableAccounts = 10
)

var defaultTestMinStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)

type MigrationModuleTestSuite struct {
	suites.MigrationModuleSuite

	// numMorseClaimableAccounts is the number of morse claimable accounts to
	// generate when calling #GenerateMorseAccountState.
	numMorseClaimableAccounts int

	appMinStake cosmostypes.Coin

	// AppServiceConfig is the service config to be used when claiming morse accounts.
	// It is assigned in the #SetupTest method.
	appServiceConfig *sharedtypes.ApplicationServiceConfig
}

func (s *MigrationModuleTestSuite) SetupTest() {
	s.ResetTestApp(defaultNumMorseClaimableAccounts, defaultTestMinStake)
}

// ResetTestApp re-runs the #SetupTest logic with the given parameters.
func (s *MigrationModuleTestSuite) ResetTestApp(
	numMorseClaimableAccounts int,
	minStake cosmostypes.Coin,
) {
	s.appMinStake = minStake

	// Set the default application min stake.
	// DEV_NOTE: This is simpler than modifying genesis or on-chain params.
	apptypes.DefaultMinStake = s.appMinStake

	// Initialize a new integration app for the suite.
	s.NewApp(s.T())

	s.numMorseClaimableAccounts = numMorseClaimableAccounts
	s.appServiceConfig = &sharedtypes.ApplicationServiceConfig{ServiceId: testServiceId}

	// Assign the app to nested suites.
	s.AppSuite.SetApp(s.GetApp())
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

// TestImportMorseClaimableAccounts_ErrorInvalidAuthority tests the error case when the authority address is invalid.
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts_ErrorInvalidAuthority() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)

	// random authority address
	invalidAuthority := sample.AccAddress()
	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		invalidAuthority,
		*s.GetAccountState(s.T()),
	)
	require.NoError(s.T(), err)

	// Import Morse claimable accounts.
	_, err = s.GetApp().RunMsg(s.T(), msgImport)

	expectedErr := migrationtypes.ErrInvalidSigner.Wrapf("invalid authority address (%s)", invalidAuthority)
	require.ErrorContains(s.T(), err, expectedErr.Error())
}

// TestImportMorseClaimableAccounts_ErrorInvalidHash tests the error case when the hash is invalid.
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts_ErrorInvalidHash() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)

	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		sample.AccAddress(), // random authority address
		*s.GetAccountState(s.T()),
	)
	require.NoError(s.T(), err)

	// Set an invalid hash.
	msgImport.MorseAccountStateHash = []byte(mockMorseAccountStateHash)

	// Import Morse claimable accounts.
	_, err = s.GetApp().RunMsg(s.T(), msgImport)

	expectedErr := migrationtypes.ErrMorseAccountsImport.Wrapf("invalid MorseAccountStateHash size")
	require.ErrorContains(s.T(), err, expectedErr.Error())
}
