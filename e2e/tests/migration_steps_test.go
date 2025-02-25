//go:build e2e && manual

package e2e

import (
	"testing"

	"github.com/regen-network/gocuke"
)

type migrationSuite struct {
	gocuke.TestingT
}

type actorTypeEnum = string

const (
	actorTypeApp      actorTypeEnum = "app"
	actorTypeSupplier actorTypeEnum = "supplier"
	actorTypeGateway  actorTypeEnum = "gateway"
)

// TestMigrationFeatures runs the migration.feature file ONLY.
// NOTE: This test has the e2e and manual build constraints because it is an E2E
// test which depends on a large Morse node snapshot being available locally.
// See: https://pocket-snapshot.liquify.com/#/pruned/
//
// To run this test use: make test_e2e_migration
// OR
// go test -v ./e2e/tests/migration_steps_test.go -tags=e2e,manual
func TestMigrationFeatures(t *testing.T) {
	gocuke.NewRunner(t, &migrationSuite{}).Path("migration_*.feature").Run()
}

func (s *migrationSuite) ALocalMorseNodePersistedStateExists() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) NoMorseclaimableaccountsExist() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountIsStakedAsAn(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonStakeEqualsTheOfTheMorseclaimableaccount(actorType actorTypeEnum, totalTokensStakePct float64) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonStakeIncreasedByTheOfTheMorseclaimableaccount(actorType actorTypeEnum, totalTokensStakePct float64) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsAnWithoutSpecifyingTheStakeAmount(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsAnWithAStakeEqualTo(a string, b string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheAuthorityExecutesWithWrittenTo(commandStr, stdioStream, outputFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorsestateexportIsWrittenTo(morseStateExportFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AnUnclaimedMorseclaimableaccountWithAKnownPrivateKeyExists() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AShannonDestinationKeyExistsInTheLocalKeyring() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheSumOfAllMorseclaimableaccountTokens() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheSumOfAndOfTheMorseclaimableaccount(balanceSummandField1, balanceSummandField2 string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonServiceConfigIsUpdatedIfApplicable(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheAuthorityExecutes(commandStr string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseaccountstateIsWrittenTo(morseAccountStateFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorseaccountstateInIsValid(morseAccountStateFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountUpoktBalanceIsNonzero() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) ThePrivateKeyIsUsedToClaimAMorseclaimableaccountAsANonactorAccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) ThePrivateKeyIsUsedToClaimAMorseclaimableaccountAsAnWithAStakeEqualTo(actorType actorTypeEnum, totalTokensStakePct float64) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheRemainingTokensOfTheMorseclaimableaccount() {
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
