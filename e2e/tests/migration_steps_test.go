//go:build e2e

package e2e

import (
	"os"
	"path"
	"strings"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	oneshotTag          = "@oneshot"
	manualTag           = "@manual"
	defaultMorseDataDir = ".pocket"
)

type migrationSuite struct {
	gocuke.TestingT
	suite

	nextMorseKeyIdx uint64
}

type actorTypeEnum = string

const (
	actorTypeApp      actorTypeEnum = "app"
	actorTypeSupplier actorTypeEnum = "supplier"
	actorTypeGateway  actorTypeEnum = "gateway"
)

var morseDatabaseFileNames = []string{
	"application.db",
	"blockstore.db",
	"evidence.db",
	"state.db",
	"txindexer.db",
}

// Before runs prior to the suite's tests.
func (s *migrationSuite) Before() {
	// DEV_NOTE: MUST assign the TestingT to the embedded suite before it is called (automatically).
	s.suite.TestingT = s.TestingT
	s.suite.Before()
}

// TestMigrationWithFixtureData runs the migration_fixture.feature file ONLY.
// To run this test use:
//
// The @oneshot tag indicates that a given feature is non-idempotent with respect
// to its impact on the network state. In such cases, a complete network reset
// is required before running these features again.
//
//	$ make test_e2e_migration_fixture
func TestMigrationWithFixtureData(t *testing.T) {
	gocuke.NewRunner(t, &migrationSuite{}).
		Path("migration_fixture.feature").
		Run()
}

// TestMigrationWithSnapshotData runs the migration_snapshot.feature file ONLY.
// NOTE: This test depends on a large Morse node snapshot being available locally.
// See: https://pocket-snapshot.liquify.com/#/pruned/
//
// To run this test use:
//
//	$ make test_e2e_migration_snapshot
func TestMigrationWithSnapshotData(t *testing.T) {
	gocuke.NewRunner(t, &migrationSuite{}).
		Path("migration_snapshot.feature").
		Run()
}

func (s *migrationSuite) ALocalMorseNodePersistedStateExists() {
	homeDir, err := os.UserHomeDir()
	require.NoError(s, err)

	// Check for the $HOME/.pocket/application.db, etc. files.
	pocketDir := path.Join(homeDir, defaultMorseDataDir)
	for _, dbFileName := range morseDatabaseFileNames {
		dbPath := path.Join(pocketDir, dbFileName)
		_, err = os.Stat(dbPath)
		require.NoErrorf(s, err, "expected %s to exist", dbPath)
	}
}

func (s *migrationSuite) NoMorseclaimableaccountsExist() {
	morseClaimableAccounts := s.QueryListMorseClaimableAccounts()
	require.Lessf(s, len(morseClaimableAccounts), 1, "expected 0 morse claimable accounts, got %d", len(morseClaimableAccounts))
}

func (s *migrationSuite) TheShannonDestinationAccountIsStakedAsAn(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonStakeIncreasedByTheOfTheMorseclaimableaccount(actorType actorTypeEnum, totalTokensStakePct float64) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsAn(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheAuthorityExecutesWithWrittenTo(commandStr, stdioStream, outputFileName string) {
	_, err := s.pocketd.RunCommand(strings.Split(commandStr, " ")...)
	require.NoError(s, err)

	var output string
	switch stdioStream {
	case "stdout":
		output = s.pocketd.result.Stdout
	case "stderr":
		output = s.pocketd.result.Stderr
	default:
		s.Fatalf("ERROR: unknown stdio stream %s", stdioStream)
	}

	s.writeTempFile(outputFileName, output)
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) writeTempFile(fileName string, content string) {
	outputPath, err := os.CreateTemp("", fileName)
	require.NoError(s, err)

	// Delete the temp file when the test completes.
	s.Cleanup(func() {
		_ = os.Remove(outputPath.Name())
	})

	_, err = outputPath.WriteString(content)
	require.NoError(s, err)
}

func (s *migrationSuite) AMorsestateexportIsWrittenTo(morseStateExportFile string) {
	morseStateExportBz, err := os.ReadFile(morseStateExportFile)
	require.NoError(s, err)

	morseStateExport := new(migrationtypes.MorseStateExport)
	err = morseStateExport.Unmarshal(morseStateExportBz)
	require.NoError(s, err)
}

func (s *migrationSuite) AnUnclaimedMorseclaimableaccountWithAKnownPrivateKeyExists() {
	// assign/increment s.nextMorseKeyIdx
	// assign s.nextMorseAccount (MorseClaimableAccount)
	idx := s.NextMorseKeyIdx()
	morseClaimableAccount, err := testmigration.GenMorseClaimableAccount(idx, testmigration.AllUnstakedMorseAccountActorType)
	require.NoError(s, err)

	// ensure MorseClaimableAccount exists on-chain
	foundMorseClaimableAccount := s.QueryShowMorseClaimableAccount(morseClaimableAccount.MorseSrcAddress)
	require.Equal(s, morseClaimableAccount, foundMorseClaimableAccount)
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) NextMorseKeyIdx() uint64 {
	s.nextMorseKeyIdx++
	return s.nextMorseKeyIdx
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) QueryShowMorseClaimableAccount(morseSrcAddress string) migrationtypes.MorseClaimableAccount {
	cmdResult, err := s.pocketd.RunCommand(
		"query",
		"migration",
		"show-morse-claimable-account",
		morseSrcAddress,
		"--output=json",
	)
	require.NoError(s, err)

	res := new(migrationtypes.QueryMorseClaimableAccountResponse)
	err = cmtjson.Unmarshal([]byte(cmdResult.Stdout), res)
	require.NoError(s, err)

	return res.MorseClaimableAccount
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) QueryListMorseClaimableAccounts() []migrationtypes.MorseClaimableAccount {
	cmdResult, err := s.pocketd.RunCommand(
		"query",
		"migration",
		"list-morse-claimable-account",
		"--output=json",
	)
	require.NoError(s, err)

	res := new(migrationtypes.QueryAllMorseClaimableAccountResponse)
	err = cmtjson.Unmarshal([]byte(cmdResult.Stdout), res)
	require.NoError(s, err)

	return res.MorseClaimableAccount
}

func (s *migrationSuite) AShannonDestinationKeyExistsInTheLocalKeyring() {
	// assign/increment s.nextShannonKeyIdx
	// check if key already exists
	// if not, generate a new key

	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheSumOfAllMorseclaimableaccountTokens() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheUnstakedBalanceAmountOfTheMorseclaimableaccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonServiceConfigIsUpdatedIfApplicable(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheAuthorityExecutes(commandStr string) {
	// TODO_IN_THIS_COMMIT: this isn't quite right, this will need to be an authz exec command...

	// DEV_NOTE: If the command doesn't start with "poktrolld"
	commandStringParts := strings.Split(commandStr, " ")
	if len(commandStringParts) < 0 && commandStringParts[0] != "poktrolld" {
		s.Fatalf("ERROR: expected a poktrolld command but got %q", commandStr)
	}

	// Remove the "poktrolld" part of the command string because
	// s.pocketd.RunCommand() provides this part of the final command string.
	commandStringParts = commandStringParts[1:]

	_, err := s.pocketd.RunCommand(commandStringParts...)
	require.NoError(s, err)
}

func (s *migrationSuite) AMorseaccountstateIsWrittenTo(morseAccountStateFile string) {
	morseAccountStateBz, err := os.ReadFile(morseAccountStateFile)
	require.NoError(s, err)

	morseAccountState := new(migrationtypes.MorseAccountState)
	err = cmtjson.Unmarshal(morseAccountStateBz, morseAccountState)
	require.NoError(s, err)
}

func (s *migrationSuite) TheMorseaccountstateInIsValid(morseAccountStateFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountUpoktBalanceIsNonzero() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorseclaimableaccountsArePersistedOnchain() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountDoesNotExistOnchain() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsANonactorAccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountExistsOnchain() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountIsNotStakedAsAn(a string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) MorsePrivateKeysAreAvailableInTheFollowingActorTypeDistribution(a gocuke.DataTable) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewApplication() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AnApplicationIsStaked() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewSupplier() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) ASupplierIsStaked() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewNonactorAccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingApplication() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingSupplier() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseaccountstateHasSuccessfullyBeenImportedWithTheFollowingClaimableAccountsTypeDistribution(a gocuke.DataTable) {
	// TODO_IN_THIS_COMMIT: this should be idempotent; check if import has already been done and skip if it has.

	// TODO_IN_THIS_COMMIT: something better...
	morseStateExportBz, _, err := testmigration.NewMorseStateExportAndAccountStateBytes(10, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(s, err)

	err = os.WriteFile("morse_state_export.json", morseStateExportBz, 0644)
	require.NoError(s, err)

	// TODO_IN_THIS_COMMIT: extract file path(s) to suite members...
	s.TheAuthorityExecutes("poktrolld tx migration collect-morse-accounts morse_state_export.json morse_account_state.json")
	s.AMorseaccountstateIsWrittenTo("morse_account_state.json")

	// TODO_IN_THIS_COMMIT:
	s.NoMorseclaimableaccountsExist()
	//s.TheMorseaccountstateInIsValid("morse_account_state.json")
	s.TheAuthorityExecutes("poktrolld tx migration import-morse-claimable-accounts morse_account_state.json")
	//s.TheMorseclaimableaccountsArePersistedOnchain()
}

func (s *migrationSuite) TheAuthoritySucessfullyImportsMorseaccountstateGeneratedFromTheSnapshotState() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingNonactorAccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseNodeSnapshotIsAvailable() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}
