//go:build e2e && oneshot

package e2e

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	// cmdUsagePattern is a substring to search for in the output of a CLI command
	// to determine whether it was unsuccessful, despite returning a zero exit code.
	cmdUsagePattern = `--help" for more`
)

var (
	// The following MUST be global variables as their state MUST NOT be reset
	// between test cases.

	// morseKeyIdx is the index of the "current" morse private key to be used.
	// It is intended to be passed to testmigration.GenMorsePrivateKey() to derive
	// (deterministically) the private key which corresponds to respective Morse
	// accounts present in the MorseAccountState fixture.
	morseKeyIdx uint64

	// shannonKeyIdx is the index of the "current" shannon private key to be used.
	// It is used to interpolate a Shannon key name string, which is used with the
	// `poktrolld keys add` command to generate a unique key which can be used to
	// sign test transactions (via the `--from` flag)
	shannonKeyIdx uint64
)

type migrationSuite struct {
	gocuke.TestingT
	suite

	// expectedNumAccounts is the number of MorseClaimableAccounts expected to be
	// imported from the MorseAccountState.
	expectedNumAccounts int

	// morseClaimableAccount is used to hold the query result for the current
	// MorseClaimableAccount in consideration.
	morseClaimableAccount *migrationtypes.MorseClaimableAccount

	// morseAccountClaimHeight is the block height at which the current
	// MorseClaimableAccount should be claimed.
	morseAccountClaimHeight int64

	// existingUnstakedBalanceUpokt is the upokt balance of the claiming (Shannon)
	// account, prior to any given test scenario. It is queried and populated in the
	// Before() method, which is called before each test case.
	existingUnstakedBalanceUpokt cosmostypes.Coin

	// faucetFundedBalanceUpokt is the upokt balance that is transferred by the faucet during setup.
	faucetFundedBalanceUpokt cosmostypes.Coin

	// expectedBalanceUpoktDiffCoin is used to hold the expected difference between the
	// claiming (Shannon) account balance before and after the test scenario.
	expectedBalanceUpoktDiffCoin cosmostypes.Coin
}

type actorTypeEnum = string

const (
	actorTypeApp      actorTypeEnum = "app"
	actorTypeSupplier actorTypeEnum = "supplier"
	actorTypeGateway  actorTypeEnum = "gateway"
)

var (
	defaultMorseDataDir    = path.Join(".pocket", "data")
	morseDatabaseFileNames = []string{
		"application.db",
		"blockstore.db",
		"evidence.db",
		"state.db",
		"txindexer.db",
	}
)

// Before runs prior to the suite's tests.
func (s *migrationSuite) Before() {
	// DEV_NOTE: MUST assign the TestingT to the embedded suite before it is called (automatically).
	s.suite.TestingT = s.TestingT
	s.suite.Before()

	// Initialize the morse & shannon key indices.
	s.nextShannonKeyIdx()
	s.nextMorseKeyIdx()

	s.existingUnstakedBalanceUpokt = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	if shannonDestAddr, isFound := s.getShannonKeyAddress(); isFound {
		if _, isFound = s.queryAccount(shannonDestAddr); isFound {
			upoktBalanceInt := s.getAccBalance(s.getShannonKeyName())
			upoktBalanceCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, int64(upoktBalanceInt))
			s.existingUnstakedBalanceUpokt = upoktBalanceCoin
		}
	}
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
	morseClaimableAccounts := s.queryListMorseClaimableAccounts()
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

func (s *migrationSuite) TheAuthorityExecutesWithStdoutWrittenTo(commandStr, outputFileName string) {
	output, err := s.runCommand(commandStr)
	require.NoError(s, err, commandStr)

	s.writeTempFile(outputFileName, output)
}

func (s *migrationSuite) AMorsestateexportIsWrittenTo(morseStateExportFile string) {
	morseStateExportBz, err := os.ReadFile(morseStateExportFile)
	require.NoError(s, err)

	morseStateExport := new(migrationtypes.MorseStateExport)
	err = cmtjson.Unmarshal(morseStateExportBz, morseStateExport)
	require.NoError(s, err)
}

func (s *migrationSuite) AnUnclaimedMorseclaimableaccountWithAKnownPrivateKeyExists() {
	// Increment s.morseKeyIdx for the next morse private key to be used.
	idx := s.nextMorseUnstakedKeyIdx()
	expectedMorseClaimableAccount, err := testmigration.GenMorseClaimableAccount(idx, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(s, err)

	// Ensure MorseClaimableAccount exists on-chain.
	foundMorseClaimableAccount := s.queryShowMorseClaimableAccount(expectedMorseClaimableAccount.MorseSrcAddress)
	require.Equal(s, expectedMorseClaimableAccount, &foundMorseClaimableAccount)

	s.expectedBalanceUpoktDiffCoin = foundMorseClaimableAccount.TotalTokens()
	s.morseClaimableAccount = &foundMorseClaimableAccount
}

func (s *migrationSuite) AShannonDestinationKeyExistsInTheLocalKeyring() {
	// assign/increment s.shannonKeyIdx
	// check if key already exists
	// if not, generate a new key
	nextKeyName := s.nextShannonKeyIdx()
	if s.keyExistsInKeyring(nextKeyName) {
		return
	}

	s.addKeyToKeyring(nextKeyName)
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheSumOfAllMorseclaimableaccountTokens() {
	currentUpoktBalanceInt := s.getAccBalance(s.getShannonKeyName())
	currentUpoktBalanceCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, int64(currentUpoktBalanceInt))
	balanceUpoktDiffCoin := currentUpoktBalanceCoin.Sub(s.existingUnstakedBalanceUpokt)

	expectedBalanceUpoktDiffCoin := s.expectedBalanceUpoktDiffCoin.Add(s.faucetFundedBalanceUpokt)
	require.Equal(s, expectedBalanceUpoktDiffCoin, balanceUpoktDiffCoin)
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheUnstakedBalanceAmountOfTheMorseclaimableaccount() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonServiceConfigIsUpdatedIfApplicable(actorType actorTypeEnum) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheAuthorityExecutes(commandStr string) {
	// DEV_NOTE: If the command doesn't start with "poktrolld"
	commandStringParts := strings.Split(commandStr, " ")
	if len(commandStringParts) < 0 && commandStringParts[0] != "poktrolld" {
		s.Fatalf("ERROR: expected a poktrolld command but got %q", commandStr)
	}

	// Remove the "poktrolld" part of the command string because
	// s.pocketd.RunCommand() provides this part of the final command string.
	commandStringParts = commandStringParts[1:]

	var (
		results *commandResult
		err     error
	)
	switch {
	case strings.Contains(commandStr, "import-morse-accounts"):
		rpcURL, err := url.Parse(defaultRPCURL)
		require.NoError(s, err)

		grpcAddrFlagString := fmt.Sprintf(
			"--grpc-addr=%s:%d",
			rpcURL.Hostname(),
			defaultGRPCPort,
		)
		grpcAddrFlagParts := strings.Split(grpcAddrFlagString, "=")
		commandStringParts = append(commandStringParts,
			"--from", "pnf",
			keyRingFlag,
			chainIdFlag,
		)
		commandStringParts = append(commandStringParts, grpcAddrFlagParts...)
		results, err = s.pocketd.RunCommandOnHost("", commandStringParts...)
	default:
		results, err = s.pocketd.RunCommand(commandStringParts...)
	}

	require.NoError(s, err)
	if strings.Contains(results.Stdout, cmdUsagePattern) {
		s.Fatalf(
			"unexpected command usage/help printed.\nCommand: %s\nStdout: %s",
			results.Command,
			results.Stdout,
		)
	}
}

func (s *migrationSuite) AMorseaccountstateIsWrittenTo(morseAccountStateFile string) {
	morseAccountStateBz, err := os.ReadFile(morseAccountStateFile)
	require.NoError(s, err)

	morseAccountState := new(migrationtypes.MorseAccountState)
	err = cmtjson.Unmarshal(morseAccountStateBz, morseAccountState)
	require.NoError(s, err)
}

func (s *migrationSuite) TheShannonDestinationAccountUpoktBalanceIsNonzero() {
	upoktBalanceAmount := s.getAccBalance(s.getShannonKeyName())
	require.Greater(s, upoktBalanceAmount, 0)
}

func (s *migrationSuite) TheMorseclaimableaccountsArePersistedOnchain() {
	morseAccounts := s.queryListMorseClaimableAccounts()
	require.Equal(s, s.expectedNumAccounts, len(morseAccounts))
}

func (s *migrationSuite) TheShannonAccountIsFundedWith(fundCoinString string) {
	fundCoin, err := cosmostypes.ParseCoinNormalized(fundCoinString)
	require.NoError(s, err)

	s.faucetFundedBalanceUpokt = fundCoin

	s.buildAddrMap()
	shannonKeyName := s.getShannonKeyName()
	shannonAddr, isFound := accNameToAddrMap[shannonKeyName]
	require.Truef(s, isFound, "key %q not found in poktrolld keyring", shannonKeyName)

	upokt, err := cosmostypes.ParseCoinNormalized(fundCoinString)
	require.NoErrorf(s, err, "unable to parse coin string %q", upokt)

	s.fundAddress(shannonAddr, upokt)
}

func (s *migrationSuite) TheShannonDestinationAccountDoesNotExistOnchain() {
	s.buildAddrMap()
	shannonDestAddr, isFound := accNameToAddrMap[s.getShannonKeyName()]
	require.Truef(s, isFound, "key %q not found in poktrolld keyring", s.getShannonKeyName())

	_, isFound = s.queryAccount(shannonDestAddr)
	require.False(s, isFound)
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsANonactorAccount() {
	// generate the deterministic fixture morse private key
	morsePrivKey := testmigration.GenMorsePrivateKey(s.getMorseKeyIdx())

	// Encrypt and write the morse private key to a file, consistent with the Morse CLI's `accounts export` command.
	privKeyArmoredJSONString, err := testmigration.EncryptArmorPrivKey(morsePrivKey, "", "")
	require.NoError(s, err)

	privKeyArmoredJSONPath := s.writeTempFile("morse_private_key.json", []byte(privKeyArmoredJSONString))

	// poktrolld tx migration claim-account --from=shannon-key-xxx <morse_src_address>
	res, err := s.pocketd.RunCommandOnHost("",
		"tx", "migration", "claim-account",
		"--from", s.getShannonKeyName(),
		keyRingFlag,
		chainIdFlag,
		"--yes",
		"--output=json",
		"--no-passphrase",
		privKeyArmoredJSONPath,
	)
	require.NoError(s, err)

	// Track the height at which the morse claimable account was claimed.
	s.morseAccountClaimHeight = s.getCurrentBlockHeight()

	if strings.Contains(res.Stdout, cmdUsagePattern) {
		s.Fatalf(
			"unexpected command usage/help printed.\nCommand: %s\nStdout: %s",
			res.Command,
			res.Stdout,
		)
	}
}

func (s *migrationSuite) TheShannonDestinationAccountExistsOnchain() {
	s.faucetFundedBalanceUpokt = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1)
	s.TheShannonAccountIsFundedWith(s.faucetFundedBalanceUpokt.String())
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

func (s *migrationSuite) AMorseaccountstateWithAccountsInADistributionHasSuccessfullyBeenImported(numAccountsStr, distributionString string) {
	var err error
	s.expectedNumAccounts, err = strconv.Atoi(numAccountsStr)
	require.NoError(s, err)

	morseAccounts := s.queryListMorseClaimableAccounts()
	switch {
	case len(morseAccounts) == s.expectedNumAccounts:
		s.Log("INFO: morse claimable accounts already imported, skipping...")
		return
	case len(morseAccounts) == 0:
		// Continue.
	default:
		s.Fatalf("expected 0 morse claimable accounts, got %d", len(morseAccounts))
	}

	var distributionFn testmigration.MorseAccountActorTypeDistributionFn
	switch distributionString {
	case "round-robin":
		distributionFn = testmigration.RoundRobinAllMorseAccountActorTypes
	default:
		s.Fatalf("unknown morse account distribution: %q", distributionString)
	}

	morseStateExportBz, _, err := testmigration.NewMorseStateExportAndAccountStateBytes(s.expectedNumAccounts, distributionFn)
	require.NoError(s, err)

	err = os.WriteFile("morse_state_export.json", morseStateExportBz, 0644)
	require.NoError(s, err)

	s.TheAuthorityExecutes("poktrolld tx migration collect-morse-accounts morse_state_export.json morse_account_state.json")
	s.AMorseaccountstateIsWrittenTo("morse_account_state.json")

	s.TheAuthorityExecutes("poktrolld tx migration import-morse-accounts morse_account_state.json")
	s.TheMorseclaimableaccountsArePersistedOnchain()
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

func (s *migrationSuite) TheMorseClaimableAccountIsMarkedAsClaimedByTheShannonAccountAtARecentBlockHeight() {
	var isShannonKeyFound bool
	expectedMorseClaimableAccount := *s.morseClaimableAccount
	expectedMorseClaimableAccount.ClaimedAtHeight = s.morseAccountClaimHeight
	expectedMorseClaimableAccount.ShannonDestAddress, isShannonKeyFound = s.getShannonKeyAddress()
	require.True(s, isShannonKeyFound)

	*s.morseClaimableAccount = s.queryShowMorseClaimableAccount(s.morseClaimableAccount.MorseSrcAddress)
	require.Equal(s, &expectedMorseClaimableAccount, s.morseClaimableAccount)
}

// runCommand executes the given command string and returns the output and any error.
func (s *migrationSuite) runCommand(commandStr string) ([]byte, error) {
	commandStringParts := strings.Split(commandStr, " ")

	cmd := exec.Command(commandStringParts[0], commandStringParts[1:]...)
	output, err := cmd.CombinedOutput()

	return output, err
}

// writeTempFile creates a temporary file with the given fileName and content.
// It returns the path to the temporary file and the temporary file is removed
// when the test completes.
func (s *migrationSuite) writeTempFile(fileName string, content []byte) string {
	outputPath, err := os.CreateTemp("", fileName)
	require.NoError(s, err)
	defer func() {
		_ = outputPath.Close()
	}()

	// Delete the temp file when the test completes.
	s.Cleanup(func() {
		_ = os.Remove(outputPath.Name())
	})

	_, err = outputPath.Write(content)
	require.NoError(s, err)

	return outputPath.Name()
}

// nextMorseKeyIdx increments the morseKeyIdx global variable and returns the
// incremented value.
func (s *migrationSuite) nextMorseKeyIdx() uint64 {
	morseKeyIdx++
	return morseKeyIdx
}

// getMorseKeyIdx returns the current value of the morseKeyIdx global variable.
func (s *migrationSuite) getMorseKeyIdx() uint64 {
	return morseKeyIdx
}

// nextMorseUnstakedKeyIdx returns the next morse private key index which is
// intended to be used for unstaked morse accounts. If the current morseKeyIdx
// is not an unstaked morse account, morseKeyIdx is incremented until the next
// Morse key index which should be an unstaked account, given the round-robin
// distribution of morse account actor types.
func (s *migrationSuite) nextMorseUnstakedKeyIdx() uint64 {
	currentIdx := s.getMorseKeyIdx()
	// Skip non-application account keys.
	for {
		if testmigration.GetRoundRobinMorseAccountActorType(currentIdx) ==
			testmigration.MorseUnstakedActor {
			break
		}
		currentIdx = s.nextMorseKeyIdx()
	}

	return currentIdx
}

// nextShannonKeyIdx randomizes the shannon key index and returns a key name
// which is derived from the new index.
func (s *migrationSuite) nextShannonKeyIdx() string {
	shannonKeyIdx = rand.Uint64()
	return s.getShannonKeyName()
}

// getShannonKeyName returns the key name derived the current shannon key index.
func (s *migrationSuite) getShannonKeyName() string {
	return fmt.Sprintf("shannon-key-%d", shannonKeyIdx)
}

// getSShannonKeyAddress checks if the key corresponding to the current shannon key index
// is present in the poktrolld keyring. If it is, it returns the address and true. Otherwise,
// it returns an empty string and false.
func (s *migrationSuite) getShannonKeyAddress() (shannonAddr string, isFound bool) {
	s.buildAddrMap()
	shannonKeyName := s.getShannonKeyName()
	shannonAddr, isFound = accNameToAddrMap[shannonKeyName]
	return shannonAddr, isFound
}

// queryShowMorseClaimableAccount queries the migration module for the MorseClaimableAccount with the given morseSrcAddress.
// It will fail the test if the account is not found.
func (s *migrationSuite) queryShowMorseClaimableAccount(morseSrcAddress string) migrationtypes.MorseClaimableAccount {
	cmdResult, err := s.pocketd.RunCommandOnHost(
		"",
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

// queryListMorseClaimableAccounts queries the migration module for all morse claimable accounts.
func (s *migrationSuite) queryListMorseClaimableAccounts() []migrationtypes.MorseClaimableAccount {
	cmdResult, err := s.pocketd.RunCommandOnHost(
		"",
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

// getCurrentBlockHeight uses poktrolld to query for the current block height.
func (s *migrationSuite) getCurrentBlockHeight() int64 {
	blockQueryRes, err := s.pocketd.RunCommandOnHost("",
		"query", "block",
		"--output", "json",
	)
	require.NoError(s, err)

	// DEV_NOTE: The first line of the response to a block query with no flag argument is not JSON:
	// "Falling back to latest block height:".
	resJSON := strings.SplitN(blockQueryRes.Stdout, "\n", 2)[1]

	// DEV_NOTE: Using jq to parse the response because cmtjson/json.Unmarshal seems
	// to be expecting hex encoded binary fields, whereas the CLI with --output json
	// seems to return base64 encoded binary fields.
	stdinBuf := new(bytes.Buffer)
	cmd := exec.Command("jq", "-r", ".header.height")
	cmd.Stdin = stdinBuf

	stdinBuf.WriteString(resJSON)
	stdout, err := cmd.Output()
	require.NoError(s, err)

	heightString := string(bytes.TrimSpace(stdout))
	height, err := strconv.Atoi(heightString)
	require.NoError(s, err)

	return int64(height)
}
