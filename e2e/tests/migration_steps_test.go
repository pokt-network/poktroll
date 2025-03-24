//go:build e2e && oneshot

package e2e

import (
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
	// between test cases, otherwise different scenarios will try to claim the
	// same Morse accounts and fail unexpectedly.
	// DEV_NOTE: Fields on gocuke suites ARE reset between scenarios.

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

	// unstakedMorseClaimableAccount is used to hold the query result for the current
	// unstaked MorseClaimableAccount in consideration.
	unstakedMorseClaimableAccount *migrationtypes.MorseClaimableAccount

	// The following indexes are used to track the next available account index
	// for each actor type.
	unstakedAccountIdx,
	appAccountIdx,
	supplierAccountIdx uint64

	// claimedActorServiceId is the service ID for which a Morse actor account is claimed.
	claimedActorServiceId string

	// appMorseClaimableAccount is used to hold the query result for the current
	// application staked MorseClaimableAccount in consideration.
	appMorseClaimableAccount *migrationtypes.MorseClaimableAccount

	// expectedMorseClaimableAccount is used to hold the query result for the
	// current expected MorseClaimableAccount in consideration.
	expectedMorseClaimableAccount *migrationtypes.MorseClaimableAccount

	// morseAccountClaimHeight is the block height at which the current
	// MorseClaimableAccount should be claimed.
	morseAccountClaimHeight int64

	// previousUnstakedBalanceUpoktOfCurrentShannonIdx is the upokt balance of the
	// claiming (Shannon) account, prior to any given test scenario. It is queried
	// and populated in the Before() method, which is called before each test case.
	previousUnstakedBalanceUpoktOfCurrentShannonIdx cosmostypes.Coin

	// previousStakedApplicationUpokt is the upokt which is staked in scenarios where an account
	// is already staked. It is initialized in the Before() method and the updated
	// in relevant subsequent steps.
	previousStakedApplicationUpokt cosmostypes.Coin

	// faucetFundedBalanceUpokt is the upokt balance that is transferred by the faucet during setup.
	faucetFundedBalanceUpokt cosmostypes.Coin
}

type actorTypeEnum = string

const (
	actorTypeApp      actorTypeEnum = "application"
	actorTypeSupplier actorTypeEnum = "supplier"
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
	// DEV_NOTE: MUST assign the TestingT to the embedded suite before it is called,
	// otherwise, the test will panic when calling methods on the embedded suite which
	// pass the receiver as a gocuke.TestingT type parameter (e.g. require.NoError(s, err)).
	s.suite.TestingT = s.TestingT
	s.suite.Before()

	// Initialize the morse & shannon key indices.
	s.nextShannonKeyIdx()
	s.nextMorseKeyIdx()

	// If the current Shannon key has an onchain balance, track it for later use in assertions.
	s.previousUnstakedBalanceUpoktOfCurrentShannonIdx = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	if shannonDestAddr, isFound := s.getShannonKeyAddress(); isFound {
		if _, isFound = s.queryAccount(shannonDestAddr); isFound {
			upoktBalanceInt := s.getAccBalance(s.getShannonKeyName())
			upoktBalanceCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, int64(upoktBalanceInt))
			s.previousUnstakedBalanceUpoktOfCurrentShannonIdx = upoktBalanceCoin
		}
	}

	// Initialize the previous actor stake here. It is updated in a relevant subsequent step.
	s.previousStakedApplicationUpokt = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
}

// TestMigrationWithFixtureData runs the migration_fixture.feature file ONLY.
// To run this test use:
//
//	$ make test_e2e_migration_fixture
//
// This feature is non-idempotent with respect to its impact on the network state.
// As a result, a complete network reset is required in-between test runs.
// A localnet reset can be performed using:
//
//	$ make localnet_down; make localnet_up
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

func (s *migrationSuite) TheShannonDestinationAccountIsStakedAsAnWithUpoktForService(actorType actorTypeEnum, upoktAmount int64, serviceId string) {
	s.previousStakedApplicationUpokt = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, upoktAmount)

	s.TheUserStakesAWithUpoktForServiceFromTheAccount(actorType, s.previousStakedApplicationUpokt.Amount.Int64(), serviceId, s.getShannonKeyName())
	s.TheUserShouldBeAbleToSeeStandardOutputContaining("txhash:")
	s.TheUserShouldBeAbleToSeeStandardOutputContaining("code: 0")
	s.ThePocketdBinaryShouldExitWithoutError()
	s.TheUserShouldWaitForSeconds(3)

	s.TheForAccountIsStakedWithUpokt(actorType, s.getShannonKeyName(), s.previousStakedApplicationUpokt.Amount.Int64())
}

func (s *migrationSuite) TheShannonDestinationAccountIsStakedAsAn(actorType actorTypeEnum) {
	switch actorType {
	case actorTypeApp:
		expectedAppStake := s.appMorseClaimableAccount.GetApplicationStake().Add(s.previousStakedApplicationUpokt)
		s.TheForAccountIsStakedWithUpokt(actorType, s.getShannonKeyName(), expectedAppStake.Amount.Int64())
	//case actorTypeSupplier:
	//  expectedSupplierStake := s.supplierMorseClaimableAccount.GetSupplierStake().Add(s.previousSupplierStakeOfCurrentShannonIdx)
	//	s.TheForAccountIsStakedWithUpokt(actorType, s.getShannonKeyName(), s.supplierMorseClaimableAccount.GetSupplierStake().Amount.Int64())
	default:
		s.Fatal("unexpected actor type %q", actorType)
	}
}

func (s *migrationSuite) TheShannonStakeIncreasedByTheCorrespondingActorStakeAmountOfTheMorseclaimableaccount(actorType actorTypeEnum) {
	actorStakeDiff := new(cosmostypes.Coin)
	expectedStakeDiff := new(cosmostypes.Coin)

	switch actorType {
	case actorTypeApp:
		currentActorStake := s.getApplicationInfo(s.getShannonKeyName()).GetStake()
		*actorStakeDiff = currentActorStake.Sub(s.previousUnstakedBalanceUpoktOfCurrentShannonIdx)
		*expectedStakeDiff = s.expectedMorseClaimableAccount.GetApplicationStake().
			Add(s.previousStakedApplicationUpokt)
	//case actorTypeSupplier:
	default:
		s.Fatal("unexpected actor type %q", actorType)
	}

	require.Equal(s, expectedStakeDiff, actorStakeDiff)
}

func (s *migrationSuite) TheMorsePrivateKeyIsUsedToClaimAMorseclaimableaccountAsAnForService(actorType actorTypeEnum, serviceId string) {
	s.claimedActorServiceId = serviceId

	var morseKeyIdx uint64
	switch actorType {
	case actorTypeApp:
		// Assign the expected claimable account for the current scenario.
		s.expectedMorseClaimableAccount = s.appMorseClaimableAccount
		morseKeyIdx = s.appAccountIdx
	default:
		s.Fatal("unexpected actor type %q", actorType)
	}

	morsePrivKey := testmigration.GenMorsePrivateKey(morseKeyIdx)

	// Encrypt and write the morse private key to a file, consistent with the Morse CLI's `accounts export` command.
	privKeyArmoredJSONString, err := testmigration.EncryptArmorPrivKey(morsePrivKey, "", "")
	require.NoError(s, err)

	privKeyArmoredJSONPath := s.writeTempFile("morse_private_key.json", []byte(privKeyArmoredJSONString))

	// poktrolld tx migration claim-application --from=shannon-key-xxx <morse_src_address> <service_id>
	res, err := s.pocketd.RunCommandOnHost("",
		"tx", "migration", "claim-application",
		"--from", s.getShannonKeyName(),
		keyRingFlag,
		chainIdFlag,
		"--yes",
		"--output=json",
		"--no-passphrase",
		privKeyArmoredJSONPath,
		serviceId,
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
	s.unstakedAccountIdx = s.nextMorseUnstakedKeyIdx()
	s.appAccountIdx = s.nextMorseApplicationKeyIdx()

	// Since this is a step which is common to usage across multiple actor types,
	// we must store the next available account index for each actor type for use
	// in subsequent steps.
	s.unstakedMorseClaimableAccount = s.getMorseClaimableAccountByIdx(s.unstakedAccountIdx)
	s.appMorseClaimableAccount = s.getMorseClaimableAccountByIdx(s.appAccountIdx)
}

// getMorseClaimableAccountByIdx returns a MorseClaimableAccount for the given index.
// It ensures that the MorseClaimableAccount exists on-chain.
func (s *migrationSuite) getMorseClaimableAccountByIdx(idx uint64) *migrationtypes.MorseClaimableAccount {
	expectedMorseClaimableAccount, err := testmigration.GenMorseClaimableAccount(idx, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(s, err)

	// Ensure MorseClaimableAccount exists on-chain.
	foundMorseClaimableAccount := s.queryShowMorseClaimableAccount(expectedMorseClaimableAccount.MorseSrcAddress)
	require.Equal(s, expectedMorseClaimableAccount, &foundMorseClaimableAccount)

	return &foundMorseClaimableAccount
}

func (s *migrationSuite) AShannonDestinationKeyExistsInTheLocalKeyring() {
	// check if key already exists
	// if not, generate a new key
	nextKeyName := s.nextShannonKeyIdx()
	if s.keyExistsInKeyring(nextKeyName) {
		return
	}

	s.addKeyToKeyring(nextKeyName)
}

func (s *migrationSuite) TheShannonDestinationAccountBalanceIsIncreasedByTheUnstakedBalanceAmountOfTheMorseclaimableaccount() {
	currentUpoktBalanceInt := s.getAccBalance(s.getShannonKeyName())
	currentUpoktBalanceCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, int64(currentUpoktBalanceInt))
	balanceUpoktDiffCoin := currentUpoktBalanceCoin.Sub(s.previousUnstakedBalanceUpoktOfCurrentShannonIdx)

	expectedBalanceUpoktDiffCoin := s.expectedMorseClaimableAccount.GetUnstakedBalance().
		Add(s.faucetFundedBalanceUpokt).
		Sub(s.previousStakedApplicationUpokt)
	require.Equal(s, expectedBalanceUpoktDiffCoin, balanceUpoktDiffCoin)
}

func (s *migrationSuite) TheShannonServiceConfigMatchesTheOneProvidedWhenClaimingTheMorseclaimableaccount(actorType actorTypeEnum) {
	switch actorType {
	case actorTypeApp:
		foundApp := s.getApplicationInfo(s.getShannonKeyName())
		require.Equal(s, s.claimedActorServiceId, foundApp.GetServiceConfigs()[0].GetServiceId())
	case actorTypeSupplier:
		s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
	default:
		s.Fatal("unexpected actor type %q", actorType)
	}
}

func (s *migrationSuite) TheAuthorityExecutes(commandStr string) {
	// DEV_NOTE: If the command doesn't start with "poktrolld" fail the test.
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

	// Check if the command returned an error despite having a zero exit code.
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
	// Assign the expected claimable account for the current scenario.
	s.expectedMorseClaimableAccount = s.unstakedMorseClaimableAccount

	// generate the deterministic fixture morse private key
	morsePrivKey := testmigration.GenMorsePrivateKey(s.unstakedAccountIdx)

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

func (s *migrationSuite) TheShannonDestinationAccountIsNotStakedAsAn(actorType actorTypeEnum) {
	switch actorType {
	case actorTypeApp:
		s.TheUserVerifiesTheForAccountIsNotStaked(actorType, s.getShannonKeyName())
	case actorTypeSupplier:
		s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
	default:
		s.Fatalf("unknown actor type %q", actorType)
	}
}

func (s *migrationSuite) MorsePrivateKeysAreAvailableInTheFollowingActorTypeDistribution(a gocuke.DataTable) {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewApplication() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AnApplicationIsStaked() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
	//s.TheForAccountIsStakedWithUpokt("application", s.getShannonKeyName(), s.unstakedMorseClaimableAccount.GetApplicationStake().Amount.Int64())
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewSupplier() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) ASupplierIsStaked() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsANewNonactorAccount() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingApplication() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingSupplier() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
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
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseAccountholderClaimsAsAnExistingNonactorAccount() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) AMorseNodeSnapshotIsAvailable() {
	s.Skip("TODO_MAINNET(@bryanchriswhite, #1034): Implement.")
}

func (s *migrationSuite) TheMorseClaimableAccountIsMarkedAsClaimedByTheShannonAccountAtARecentBlockHeight() {
	var isShannonKeyFound bool
	expectedMorseClaimableAccount := *s.unstakedMorseClaimableAccount
	expectedMorseClaimableAccount.ClaimedAtHeight = s.morseAccountClaimHeight
	expectedMorseClaimableAccount.ShannonDestAddress, isShannonKeyFound = s.getShannonKeyAddress()
	require.True(s, isShannonKeyFound)

	*s.unstakedMorseClaimableAccount = s.queryShowMorseClaimableAccount(s.unstakedMorseClaimableAccount.MorseSrcAddress)
	require.Equal(s, &expectedMorseClaimableAccount, s.unstakedMorseClaimableAccount)
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
	return s.nextMorseActorKeyIdx(testmigration.MorseUnstakedActor)
}

// nextMorseApplicationKeyIdx returns the next morse private key index which is
// intended to be used for staked application morse accounts. If the current
// morseKeyIdx is not a staked application morse account, morseKeyIdx is incremented
// until the next Morse key index which should be a staked application account,
// given the round-robin distribution of morse account actor types.
func (s *migrationSuite) nextMorseApplicationKeyIdx() uint64 {
	return s.nextMorseActorKeyIdx(testmigration.MorseApplicationActor)
}

// nextMorseActorKeyIdx returns the next morse private key index which matches
// the given actor type. If the current morseKeyIdx does not match, morseKeyIdx
// is incremented until the next Morse key index which does.
func (s *migrationSuite) nextMorseActorKeyIdx(actorType testmigration.MorseAccountActorType) uint64 {
	currentIdx := s.getMorseKeyIdx()
	// Skip non-matching account keys.
	for {
		if testmigration.GetRoundRobinMorseAccountActorType(currentIdx) ==
			actorType {
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

// getShannonKeyAddress checks if the key corresponding to the current shannon key index
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
