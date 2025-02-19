package migration

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testServiceId = "svc1"

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
	s.appMinStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)

	// Set the default application min stake.
	// DEV_NOTE: This is simpler than modifying genesis or on-chain params.
	apptypes.DefaultMinStake = s.appMinStake

	// Initialize a new integration app for the suite.
	s.NewApp(s.T())

	s.numMorseClaimableAccounts = 10
	s.appServiceConfig = &sharedtypes.ApplicationServiceConfig{ServiceId: testServiceId}

	// Assign the app to nested suites.
	s.AppSuite.SetApp(s.GetApp())
}

func TestMigrationModuleSuite(t *testing.T) {
	suite.Run(t, &MigrationModuleTestSuite{})
}

// TestImportMorseClaimableAccounts tests claiming of morse claimable accounts.
// It only claims account balances and does not test staking any actors as a result of claiming.
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	msgImportRes := s.ImportMorseClaimableAccounts(s.T())
	morseAccountStateHash, err := s.GetAccountState(s.T()).GetHash()
	require.NoError(s.T(), err)

	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   morseAccountStateHash,
		NumAccounts: uint64(s.numMorseClaimableAccounts),
	}
	require.Equal(s.T(), expectedMsgImportRes, msgImportRes)
}
