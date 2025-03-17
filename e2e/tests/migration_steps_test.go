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
	// TODO_IN_THIS_COMMIT: godoc...
	cmdUsagePattern = `--help" for more`
)

var (
	// TODO_IN_THIS_COMMIT: godoc... MUST be global variables...
	morseKeyIdx   uint64
	shannonKeyIdx uint64
)

type migrationSuite struct {
	gocuke.TestingT
	suite

	// TODO_IN_THIS_COMMIT: godoc...
	morseClaimableAccount *migrationtypes.MorseClaimableAccount

	// TODO_IN_THIS_COMMIT: godoc...
	morseAccountClaimHeight int64

	// TODO_IN_THIS_COMMIT: godoc... queried/populated in Before()...
	existingUnstakedBalanceUpokt cosmostypes.Coin
	// TODO_IN_THIS_COMMIT: godoc... the amount transferred by the faucet during setup/given steps...
	faucetFundedBalanceUpokt cosmostypes.Coin
	// TODO_IN_THIS_COMMIT: godoc...
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

func (s *migrationSuite) TheAuthorityExecutesWithStdoutWrittenTo(commandStr, outputFileName string) {
	// TODO_IN_THIS_COMMIT: something to more clearly indicate that this step is distinct a poktrolld sub-command invocation.
	//_, err := s.pocketd.RunCommand(strings.Split(commandStr, " ")...)
	//require.NoError(s, err)

	output, err := s.runCommand(commandStr)
	//require.NoErrorf(s, err, commandStr, string(output))
	//require.NoError(s, err, string(output))
	require.NoError(s, err, commandStr)

	s.writeTempFile(outputFileName, output)
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) runCommand(commandStr string) ([]byte, error) {
	commandStringParts := strings.Split(commandStr, " ")

	cmd := exec.Command(commandStringParts[0], commandStringParts[1:]...)
	output, err := cmd.CombinedOutput()

	return output, err
}

// TODO_IN_THIS_COMMIT: godoc and move...
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

func (s *migrationSuite) AMorsestateexportIsWrittenTo(morseStateExportFile string) {
	morseStateExportBz, err := os.ReadFile(morseStateExportFile)
	require.NoError(s, err)

	morseStateExport := new(migrationtypes.MorseStateExport)
	err = cmtjson.Unmarshal(morseStateExportBz, morseStateExport)
	require.NoError(s, err)
}

func (s *migrationSuite) AnUnclaimedMorseclaimableaccountWithAKnownPrivateKeyExists() {
	// assign/increment s.morseKeyIdx
	// assign s.nextMorseAccount (MorseClaimableAccount)
	idx := s.nextMorseUnstakedKeyIdx()
	expectedMorseClaimableAccount, err := testmigration.GenMorseClaimableAccount(idx, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(s, err)

	// ensure MorseClaimableAccount exists on-chain
	foundMorseClaimableAccount := s.QueryShowMorseClaimableAccount(expectedMorseClaimableAccount.MorseSrcAddress)
	require.Equal(s, expectedMorseClaimableAccount, &foundMorseClaimableAccount)

	s.expectedBalanceUpoktDiffCoin = foundMorseClaimableAccount.TotalTokens()
	s.morseClaimableAccount = &foundMorseClaimableAccount
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) nextMorseKeyIdx() uint64 {
	morseKeyIdx++
	return morseKeyIdx
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) getMorseKeyIdx() uint64 {
	return morseKeyIdx
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) nextMorseUnstakedKeyIdx() uint64 {
	currentIdx := s.getMorseKeyIdx()
	// Skip non-application account keys.
	for {
		if testmigration.GetMorseAccountActorType(currentIdx) ==
			testmigration.MorseUnstakedActor {
			break
		}
		currentIdx = s.nextMorseKeyIdx()
	}

	return currentIdx
}

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) QueryShowMorseClaimableAccount(morseSrcAddress string) migrationtypes.MorseClaimableAccount {
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

// TODO_IN_THIS_COMMIT: godoc and move...
func (s *migrationSuite) QueryListMorseClaimableAccounts() []migrationtypes.MorseClaimableAccount {
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

// TODO_IN_THIS_COMMIT: godoc & move...
func (s *migrationSuite) nextShannonKeyIdx() string {
	shannonKeyIdx = rand.Uint64()
	return s.getShannonKeyName()
}

// TODO_IN_THIS_COMMIT: godoc & move...
func (s *migrationSuite) getShannonKeyName() string {
	return fmt.Sprintf("shannon-key-%d", shannonKeyIdx)
}

// TODO_IN_THIS_COMMIT: godoc & move...
func (s *migrationSuite) getShannonKeyAddress() (shannonAddr string, isFound bool) {
	s.buildAddrMap()
	shannonKeyName := s.getShannonKeyName()
	shannonAddr, isFound = accNameToAddrMap[shannonKeyName]
	return shannonAddr, isFound
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheSumOfAllMorseclaimableaccountTokens() {
	currentUpoktBalanceInt := s.getAccBalance(s.getShannonKeyName())
	currentUpoktBalanceCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, int64(currentUpoktBalanceInt))
	balanceUpoktDiffCoin := currentUpoktBalanceCoin.Sub(s.existingUnstakedBalanceUpokt)

	expectedBalanceUpoktDiffCoin := s.expectedBalanceUpoktDiffCoin.Add(s.faucetFundedBalanceUpokt)
	require.Equal(s, expectedBalanceUpoktDiffCoin, balanceUpoktDiffCoin)

	//// TODO_IN_THIS_COMMIT: Reconcile "balance is increased" with the fact that we're currently checking the exact balance.
	////expectedUpoktAmount := testmigration.GenMorseUnstakedBalanceAmount(s.getMorseKeyIdx())
	//expectedUpoktAmount := s.morseClaimableAccount.UnstakedBalance.Amount.Int64()
	//// TODO_MAINNET: Remove this adjustment once the signer fee issue is resolved.
	//expectedUpoktAmount = expectedUpoktAmount + 1
	//upoktAmount := int64(s.getAccBalance(s.getShannonKeyName()))
	//require.Equal(s, expectedUpoktAmount, upoktAmount)
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

func (s *migrationSuite) TheMorseaccountstateInIsValid(morseAccountStateFile string) {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheShannonDestinationAccountUpoktBalanceIsNonzero() {
	upoktBalanceAmount := s.getAccBalance(s.getShannonKeyName())
	require.Greater(s, upoktBalanceAmount, 0)
}

func (s *migrationSuite) TheMorseclaimableaccountsArePersistedOnchain() {
	s.Skip("TODO_UPNEXT(@bryanchriswhite, #1034): Implement.")
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
	//morsePrivKey := testmigration.GenMorsePrivateKey(s.nextMorseUnstakedKeyIdx())
	morsePrivKey := testmigration.GenMorsePrivateKey(s.getMorseKeyIdx())

	// Encrypt and write the morse private key to a file, consistent with the Morse CLI's `accounts export` command.
	privKeyArmoredJSONString, err := testmigration.EncryptArmorPrivKey(morsePrivKey, "", "")
	require.NoError(s, err)

	//s.Logf("XX| %s |XX", privKeyArmoredJSONString)

	// TODO_IN_THIS_COMMIT: consolidate with any other temp file tracking pattern.
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

	// TODO_IN_THIS_COMMIT: zero exit code error handling...
	s.Logf("RESULT: %s", res.Stdout)
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

func (s *migrationSuite) AMorseaccountstateHasSuccessfullyBeenImportedWithTheFollowingClaimableAccountsTypeDistribution(a gocuke.DataTable) {
	// TODO_IN_THIS_COMMIT: this should be idempotent; check if import has already been done and skip if it has.
	morseAccounts := s.QueryListMorseClaimableAccounts()
	if len(morseAccounts) > 0 {
		s.Log("INFO: morse claimable accounts already imported, skipping...")
		return
	}

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
	s.TheAuthorityExecutes("poktrolld tx migration import-morse-accounts morse_account_state.json")
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

func (s *migrationSuite) TheMorseClaimableAccountIsMarkedAsClaimedByTheShannonAccountAtARecentBlockHeight() {
	var isShannonKeyFound bool
	expectedMorseClaimableAccount := *s.morseClaimableAccount
	expectedMorseClaimableAccount.ClaimedAtHeight = s.morseAccountClaimHeight
	expectedMorseClaimableAccount.ShannonDestAddress, isShannonKeyFound = s.getShannonKeyAddress()
	require.True(s, isShannonKeyFound)

	*s.morseClaimableAccount = s.QueryShowMorseClaimableAccount(s.morseClaimableAccount.MorseSrcAddress)
	require.Equal(s, &expectedMorseClaimableAccount, s.morseClaimableAccount)
}

// TODO_IN_THIS_COMMIT: godoc...
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

	//jqStdout, err := s.runCommand(" jq -r .header.height")
	//require.NoError(s, err)

	heightString := string(bytes.TrimSpace(stdout))
	height, err := strconv.Atoi(heightString)
	require.NoError(s, err)

	return int64(height)
}
