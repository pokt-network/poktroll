package migration

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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

	// minStake is used to set the on-chain min stake for the application & supplier modules.
	minStake cosmostypes.Coin

	// appServiceConfig is the service config to be used when claiming morse accounts as applications.
	// It is assigned in the #SetupTest method.
	appServiceConfig *sharedtypes.ApplicationServiceConfig

	// supplierServices is the service config to be used when claiming morse accounts as suppliers.
	// It is assigned in the #SetupTest method.
	supplierServices []*sharedtypes.SupplierServiceConfig
}

func (s *MigrationModuleTestSuite) SetupTest() {
	s.ResetTestApp(defaultNumMorseClaimableAccounts, defaultTestMinStake)
}

// ResetTestApp re-runs the #SetupTest logic with the given parameters.
func (s *MigrationModuleTestSuite) ResetTestApp(
	numMorseClaimableAccounts int,
	minStake cosmostypes.Coin,
) {
	s.minStake = minStake

	// Set the default application & supplier min stakes.
	// DEV_NOTE: This is simpler than modifying genesis or on-chain params.
	apptypes.DefaultMinStake = s.minStake
	suppliertypes.DefaultMinStake = s.minStake

	// Initialize a new integration app for the suite.
	s.NewApp(s.T())

	s.numMorseClaimableAccounts = numMorseClaimableAccounts
	s.appServiceConfig = &sharedtypes.ApplicationServiceConfig{ServiceId: testServiceId}
	s.supplierServices = []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: testServiceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://test.example:1234",
					RpcType: sharedtypes.RPCType_JSON_RPC,
				},
			},
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            sample.AccAddress(),
					RevSharePercentage: 100,
				},
			},
		},
	}

	// Assign the app to nested suites.
	s.ServiceSuite.SetApp(s.GetApp())
	s.AppSuite.SetApp(s.GetApp())
	s.SupplierSuite.SetApp(s.GetApp())
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
	s.NoError(err)

	expectedMsgImportRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   morseAccountStateHash,
		NumAccounts: uint64(s.numMorseClaimableAccounts),
	}
	s.Equal(expectedMsgImportRes, msgImportRes)

	foundMorseClaimableAccounts := s.QueryAllMorseClaimableAccounts(s.T())
	s.Equal(s.numMorseClaimableAccounts, len(foundMorseClaimableAccounts))

	for _, expectedMorseClaimableAccount := range morseAccountState.Accounts {
		isFound := false
		for _, foundMorseClaimableAccount := range foundMorseClaimableAccounts {
			if foundMorseClaimableAccount.GetMorseSrcAddress() == expectedMorseClaimableAccount.GetMorseSrcAddress() {
				s.Equal(*expectedMorseClaimableAccount, foundMorseClaimableAccount)
				isFound = true
				break
			}
		}
		s.True(isFound)
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
	s.NoError(err)

	// Import Morse claimable accounts.
	_, err = s.GetApp().RunMsg(s.T(), msgImport)

	expectedErr := migrationtypes.ErrInvalidSigner.Wrapf("invalid authority address (%s)", invalidAuthority)
	s.ErrorContains(err, expectedErr.Error())
}

// TestImportMorseClaimableAccounts_ErrorInvalidHash tests the error case when the hash is invalid.
func (s *MigrationModuleTestSuite) TestImportMorseClaimableAccounts_ErrorInvalidHash() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)

	msgImport, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		sample.AccAddress(), // random authority address
		*s.GetAccountState(s.T()),
	)
	s.NoError(err)

	// Set an invalid hash.
	msgImport.MorseAccountStateHash = []byte(mockMorseAccountStateHash)

	// Import Morse claimable accounts.
	_, err = s.GetApp().RunMsg(s.T(), msgImport)

	expectedErr := migrationtypes.ErrMorseAccountsImport.Wrapf("invalid MorseAccountStateHash size")
	s.ErrorContains(err, expectedErr.Error())
}
