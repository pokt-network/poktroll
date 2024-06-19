//go:build e2e

package e2e

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	sdklog "cosmossdk.io/log"
	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/yaml"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	numQueryRetries = uint8(3)
)

var (
	addrRe          *regexp.Regexp
	amountRe        *regexp.Regexp
	addrAndAmountRe *regexp.Regexp

	accNameToAddrMap     = make(map[string]string)
	accAddrToNameMap     = make(map[string]string)
	accNameToAppMap      = make(map[string]apptypes.Application)
	accNameToSupplierMap = make(map[string]sharedtypes.Supplier)

	flagFeaturesPath string
	keyRingFlag      = "--keyring-backend=test"
	chainIdFlag      = "--chain-id=poktroll"
	appGateServerUrl = "http://localhost:42069" // Keeping localhost by default because that is how we run the tests on our machines locally
)

func init() {
	addrRe = regexp.MustCompile(`address:\s+(\S+)\s+name:\s+(\S+)`)
	amountRe = regexp.MustCompile(`amount:\s+"(.+?)"\s+denom:\s+upokt`)
	addrAndAmountRe = regexp.MustCompile(`(?s)address: ([\w\d]+).*?stake:\s*amount: "(\d+)"`)

	flag.StringVar(&flagFeaturesPath, "features-path", "*.feature", "Specifies glob paths for the runner to look up .feature files")

	// If "APPGATE_SERVER_URL" envar is present, use it for appGateServerUrl
	if url := os.Getenv("APPGATE_SERVER_URL"); url != "" {
		appGateServerUrl = url
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.Printf("Running features matching %q", path.Join("e2e", "tests", flagFeaturesPath))
	m.Run()
}

type suite struct {
	gocuke.TestingT
	ctx  context.Context
	once sync.Once
	// TODO_TECHDEBT: rename to `poktrolld`.
	pocketd          *pocketdBin
	scenarioState    map[string]any // temporary state for each scenario
	cdc              codec.Codec
	proofQueryClient prooftypes.QueryClient

	// See the Cosmo SDK authz module for references related to `granter` and `grantee`
	// https://docs.cosmos.network/main/build/modules/authz
	granterName string
	granteeName string

	// moduleParamsMap is a map of module names to a map of parameter names to parameter values & types.
	expectedModuleParams moduleParamsMap
}

func (s *suite) Before() {
	s.ctx = context.Background()
	s.pocketd = new(pocketdBin)
	s.scenarioState = make(map[string]any)
	deps := depinject.Configs(
		app.AppConfig(),
		depinject.Supply(
			sdklog.NewTestLogger(s),
		),
	)
	err := depinject.Inject(deps, &s.cdc)
	require.NoError(s, err)
	s.buildAddrMap()
	s.buildAppMap()
	s.buildSupplierMap()

	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	s.proofQueryClient = prooftypes.NewQueryClient(clientCtx)
}

// TestFeatures runs the e2e tests specified in any .features files in this directory
// * This test suite assumes that a LocalNet is running
func TestFeatures(t *testing.T) {
	gocuke.NewRunner(t, &suite{}).Path(flagFeaturesPath).Run()
}

// TODO_TECHDEBT: rename `pocketd` to `poktrolld`.
func (s *suite) TheUserHasThePocketdBinaryInstalled() {
	s.TheUserRunsTheCommand("help")
}

func (s *suite) ThePocketdBinaryShouldExitWithoutError() {
	require.NoError(s, s.pocketd.result.Err)
}

func (s *suite) TheUserRunsTheCommand(cmd string) {
	cmds := strings.Split(cmd, " ")
	res, err := s.pocketd.RunCommand(cmds...)
	require.NoError(s, err, "error running command %s", cmd)
	s.pocketd.result = res
}

func (s *suite) TheUserShouldBeAbleToSeeStandardOutputContaining(arg1 string) {
	require.Contains(s, s.pocketd.result.Stdout, arg1)
}

func (s *suite) TheUserSendsUpoktFromAccountToAccount(amount int64, accName1, accName2 string) {
	args := []string{
		"tx",
		"bank",
		"send",
		accNameToAddrMap[accName1],
		accNameToAddrMap[accName2],
		fmt.Sprintf("%dupokt", amount),
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "error sending upokt from %q to %q", accName1, accName2)
	s.pocketd.result = res
}

func (s *suite) TheAccountHasABalanceGreaterThanUpokt(accName string, amount int64) {
	bal := s.getAccBalance(accName)
	require.Greaterf(s, bal, int(amount), "account %s does not have enough upokt", accName)
	s.scenarioState[accBalanceKey(accName)] = bal // save the balance for later
}

func (s *suite) AnAccountExistsFor(accName string) {
	bal := s.getAccBalance(accName)
	s.scenarioState[accBalanceKey(accName)] = bal // save the balance for later
}

func (s *suite) TheStakeOfShouldBeUpoktThanBefore(actorType string, accName string, expectedStakeChange int64, condition string) {
	// Get previous stake
	stakeKey := accStakeKey(actorType, accName)
	prevStakeAny, ok := s.scenarioState[stakeKey]
	require.True(s, ok, "no previous stake found for %s", accName)
	prevStake, ok := prevStakeAny.(int)
	require.True(s, ok, "previous stake for %s is not an int", accName)

	// Get current stake
	currStake, ok := s.getStakedAmount(actorType, accName)
	require.True(s, ok, "no current stake found for %s", accName)
	s.scenarioState[stakeKey] = currStake // save the stake for later

	// Validate the change in stake
	s.validateAmountChange(prevStake, currStake, expectedStakeChange, accName, condition, "stake")
}

func (s *suite) TheAccountBalanceOfShouldBeUpoktThanBefore(accName string, expectedStakeChange int64, condition string) {
	// Get previous balance
	balanceKey := accBalanceKey(accName)
	prevBalanceAny, ok := s.scenarioState[balanceKey]
	require.True(s, ok, "no previous balance found for %s", accName)
	prevBalance, ok := prevBalanceAny.(int)
	require.True(s, ok, "previous balance for %s is not an int", accName)

	// Get current balance
	currBalance := s.getAccBalance(accName)
	s.scenarioState[balanceKey] = currBalance // save the balance for later

	// Validate the change in stake
	s.validateAmountChange(prevBalance, currBalance, expectedStakeChange, accName, condition, "balance")
}

func (s *suite) TheUserShouldWaitForSeconds(dur int64) {
	time.Sleep(time.Duration(dur) * time.Second)
}

func (s *suite) TheUserStakesAWithUpoktFromTheAccount(actorType string, amount int64, accName string) {
	// Create a temporary config file
	configPathPattern := fmt.Sprintf("%s_stake_config_*.yaml", accName)
	configFile, err := os.CreateTemp("", configPathPattern)
	require.NoError(s, err, "error creating config file in %q", path.Join(os.TempDir(), configPathPattern))

	configContent := fmt.Sprintf(`stake_amount: %d upokt`, amount)
	_, err = configFile.Write([]byte(configContent))
	require.NoError(s, err, "error writing config file %q", configFile.Name())

	args := []string{
		"tx",
		actorType,
		fmt.Sprintf("stake-%s", actorType),
		"--config",
		configFile.Name(),
		"--from",
		accName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)

	// Remove the temporary config file
	err = os.Remove(configFile.Name())
	require.NoError(s, err, "error removing config file %q", configFile.Name())

	s.pocketd.result = res
}

func (s *suite) TheUserStakesAWithUpoktForServiceFromTheAccount(actorType string, amount int64, serviceId, accName string) {
	// Create a temporary config file
	configPathPattern := fmt.Sprintf("%s_stake_config.yaml", accName)
	configFile, err := os.CreateTemp("", configPathPattern)
	require.NoError(s, err, "error creating config file in %q", path.Join(os.TempDir(), configPathPattern))

	// Write the config content to the file
	configContent := s.getConfigFileContent(amount, actorType, serviceId)
	_, err = configFile.Write([]byte(configContent))
	require.NoError(s, err, "error writing config file %q", configFile.Name())

	// Prepare the command arguments
	args := []string{
		"tx",
		actorType,
		fmt.Sprintf("stake-%s", actorType),
		"--config",
		configFile.Name(),
		"--from",
		accName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)

	// Remove the temporary config file
	err = os.Remove(configFile.Name())
	require.NoError(s, err, "error removing config file %q", configFile.Name())

	s.pocketd.result = res
}

func (s *suite) getConfigFileContent(amount int64, actorType, serviceId string) string {
	var configContent string
	switch actorType {
	case "application":
		configContent = fmt.Sprintf(`
		stake_amount: %dupokt
		service_ids:
		  - %s`,
			amount, serviceId)
	case "supplier":
		configContent = fmt.Sprintf(`
			stake_amount: %dupokt
			services:
			  - service_id: %s
			    endpoints:
			    - publicly_exposed_url: http://relayminer:8545
			      rpc_type: json_rpc`,
			amount, serviceId)
	default:
		s.Fatalf("unknown actor type %s", actorType)
	}
	fmt.Println(yaml.NormalizeYAMLIndentation(configContent))
	return yaml.NormalizeYAMLIndentation(configContent)
}

func (s *suite) TheUserUnstakesAFromTheAccount(actorType string, accName string) {
	args := []string{
		"tx",
		actorType,
		fmt.Sprintf("unstake-%s", actorType),
		"--from",
		accName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "error unstaking %s", actorType)

	s.pocketd.result = res
}

func (s *suite) TheAccountForIsStaked(actorType, accName string) {
	stakeAmount, ok := s.getStakedAmount(actorType, accName)
	require.Truef(s, ok, "account %s of type %s SHOULD be staked", accName, actorType)
	s.scenarioState[accStakeKey(actorType, accName)] = stakeAmount // save the stakeAmount for later
}

func (s *suite) TheForAccountIsNotStaked(actorType, accName string) {
	_, ok := s.getStakedAmount(actorType, accName)
	require.Falsef(s, ok, "account %s of type %s SHOULD NOT be staked", accName, actorType)
}

func (s *suite) TheForAccountIsStakedWithUpokt(actorType, accName string, amount int64) {
	stakeAmount, ok := s.getStakedAmount(actorType, accName)
	require.Truef(s, ok, "account %s of type %s SHOULD be staked", accName, actorType)
	require.Equalf(s, int64(stakeAmount), amount, "account %s stake amount is not %d", accName, amount)
	s.scenarioState[accStakeKey(actorType, accName)] = stakeAmount // save the stakeAmount for later
}

func (s *suite) TheApplicationIsStakedForService(appName string, serviceId string) {
	for _, serviceConfig := range accNameToAppMap[appName].ServiceConfigs {
		if serviceConfig.Service.Id == serviceId {
			return
		}
	}
	s.Fatalf("application %s is not staked for service %s", appName, serviceId)
}

func (s *suite) TheSupplierIsStakedForService(supplierName string, serviceId string) {
	for _, serviceConfig := range accNameToSupplierMap[supplierName].Services {
		if serviceConfig.Service.Id == serviceId {
			return
		}
	}
	s.Fatalf("supplier %s is not staked for service %s", supplierName, serviceId)
}

func (s *suite) TheSessionForApplicationAndServiceContainsTheSupplier(appName string, serviceId string, supplierName string) {
	app, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

	expectedSupplier, ok := accNameToSupplierMap[supplierName]
	require.True(s, ok, "supplier %s not found", supplierName)

	argsAndFlags := []string{
		"query",
		"session",
		"get-session",
		app.Address,
		serviceId,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err, "error getting session for app %s and service %q", appName, serviceId)

	var resp sessiontypes.QueryGetSessionResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	for _, supplier := range resp.Session.Suppliers {
		if supplier.Address == expectedSupplier.Address {
			return
		}
	}
	s.Fatalf("session for app %s and service %s does not contain supplier %s", appName, serviceId, supplierName)
}

func (s *suite) TheApplicationSendsTheSupplierARequestForServiceWithData(appName, supplierName, serviceId, requestData string) {
	// TODO_HACK: We need to support a non self_signing LocalNet AppGateServer
	// that allows any application to send a relay in LocalNet and our E2E Tests.
	require.Equal(s, "app1", appName, "TODO_HACK: The LocalNet AppGateServer is self_signing and only supports app1.")

	res, err := s.pocketd.RunCurl(appGateServerUrl, serviceId, requestData)
	require.NoError(s, err, "error sending relay request from app %q to supplier %q for service %q", appName, supplierName, serviceId)

	relayKey := relayReferenceKey(appName, supplierName)
	s.scenarioState[relayKey] = res.Stdout
}

func (s *suite) TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName string, supplierName string) {
	// TODO_HACK: We need to support a non self_signing LocalNet AppGateServer
	// that allows any application to send a relay in LocalNet and our E2E Tests.
	require.Equal(s, "app1", appName, "TODO_HACK: The LocalNet AppGateServer is self_signing and only supports app1.")

	relayKey := relayReferenceKey(appName, supplierName)
	stdout, ok := s.scenarioState[relayKey]
	require.Truef(s, ok, "no relay response found for %s", relayKey)
	require.Contains(s, stdout, `"result":"0x`)
}

// TODO_TECHDEBT: Factor out the common logic between this step and waitForTxResultEvent,
// it is actually not possible since the later is getting the query client from
// s.scenarioState which seems to cause query client to fail with EOF error.
func (s *suite) AModuleEventIsBroadcasted(module, event string) {
	ctx, done := context.WithCancel(context.Background())

	// Construct an events query client to listen for tx events from the supplier.
	eventType := fmt.Sprintf("poktroll.%s.Event%s", module, event)
	deps := depinject.Supply(events.NewEventsQueryClient(testclient.CometLocalWebsocketURL))
	onChainClaimEventsReplayClient, err := events.NewEventsReplayClient[*block.CometNewBlockEvent](
		ctx,
		deps,
		newBlockEventSubscriptionQuery,
		block.UnmarshalNewBlockEvent,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*block.CometNewBlockEvent](
		ctx, onChainClaimEventsReplayClient.EventsSequence(ctx),
		func(_ context.Context, newBlockEvent *block.CometNewBlockEvent) {
			if newBlockEvent == nil {
				return
			}

			// Range over each event's attributes to find the "action" attribute
			// and compare its value to that of the action provided.
			for _, event := range newBlockEvent.Data.Value.ResultFinalizeBlock.Events {
				// Checks on the event. For example, for a Claim Settlement event,
				// we can parse the claim and verify the compute units.
				if event.Type == eventType {
					done()
					return
				}
			}
		},
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("timed out waiting for message with action %q", eventType)
	case <-ctx.Done():
		s.Log("Success; message detected before timeout.")
	}
}

func (s *suite) getStakedAmount(actorType, accName string) (int, bool) {
	s.Helper()
	args := []string{
		"query",
		actorType,
		fmt.Sprintf("list-%s", actorType),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting %s", actorType)
	s.pocketd.result = res

	escapedAddress := accNameToAddrMap[accName]
	amount := 0
	if strings.Contains(res.Stdout, escapedAddress) {
		matches := addrAndAmountRe.FindAllStringSubmatch(res.Stdout, -1)
		if len(matches) < 1 {
			return 0, false
		}
		for _, match := range matches {
			if match[1] == escapedAddress {
				amount, err = strconv.Atoi(match[2])
				require.NoError(s, err)
				return amount, true
			}
		}
	}
	return 0, false
}

func (s *suite) buildAddrMap() {
	s.Helper()
	res, err := s.pocketd.RunCommand(
		"keys", "list", keyRingFlag,
	)
	require.NoError(s, err, "error getting keys")
	s.pocketd.result = res
	matches := addrRe.FindAllStringSubmatch(res.Stdout, -1)
	for _, match := range matches {
		name := match[2]
		address := match[1]
		accNameToAddrMap[name] = address
		accAddrToNameMap[address] = name
	}
}

func (s *suite) buildAppMap() {
	s.Helper()
	argsAndFlags := []string{
		"query",
		"application",
		"list-application",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err, "error getting application list")
	s.pocketd.result = res
	var resp apptypes.QueryAllApplicationsResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	for _, app := range resp.Applications {
		accNameToAppMap[accAddrToNameMap[app.Address]] = app
	}
}

func (s *suite) buildSupplierMap() {
	s.Helper()
	argsAndFlags := []string{
		"query",
		"supplier",
		"list-supplier",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err, "error getting supplier list")
	s.pocketd.result = res
	var resp suppliertypes.QueryAllSuppliersResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	for _, supplier := range resp.Supplier {
		accNameToSupplierMap[accAddrToNameMap[supplier.Address]] = supplier
	}
}

// TODO_TECHDEBT(@bryanchriswhite): Cleanup & deduplicate the code related
// to this accessors. Ref: https://github.com/pokt-network/poktroll/pull/448/files#r1547930911
func (s *suite) getAccBalance(accName string) int {
	s.Helper()

	args := []string{
		"query",
		"bank",
		"balances",
		accNameToAddrMap[accName],
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting balance")
	s.pocketd.result = res

	match := amountRe.FindStringSubmatch(res.Stdout)
	require.GreaterOrEqual(s, len(match), 2, "no balance found for %s", accName)

	accBalance, err := strconv.Atoi(match[1])
	require.NoError(s, err)

	return accBalance
}

// validateAmountChange validates if the balance of an account has increased or decreased by the expected amount
func (s *suite) validateAmountChange(prevAmount, currAmount int, expectedAmountChange int64, accName, condition, balanceType string) {
	deltaAmount := int64(math.Abs(float64(currAmount - prevAmount)))
	// Verify if balance is more or less than before
	switch condition {
	case "more":
		require.GreaterOrEqual(s, currAmount, prevAmount, "%s %s expected to have more upokt but actually had less", accName, balanceType)
		require.Equal(s, expectedAmountChange, deltaAmount, "%s %s expected increase in upokt was incorrect", accName, balanceType)
	case "less":
		require.LessOrEqual(s, currAmount, prevAmount, "%s %s expected to have less upokt but actually had more", accName, balanceType)
		require.Equal(s, expectedAmountChange, deltaAmount, "%s %s expected) decrease in upokt was incorrect", accName, balanceType)
	default:
		s.Fatalf("unknown condition %s", condition)
	}

}

// TODO_IMPROVE: use `sessionId` and `supplierName` since those are the two values
// used to create the primary composite key on-chain to uniquely distinguish relays.
func relayReferenceKey(appName, supplierName string) string {
	return fmt.Sprintf("%s/%s", appName, supplierName)
}

// accBalanceKey is a helper function to create a key to store the balance
// for accName in the context of a scenario state.
func accBalanceKey(accName string) string {
	return fmt.Sprintf("balance/%s", accName)
}

// accStakeKey is a helper function to create a key to store the stake
// for accName of type actorType in the context of a scenario state.
func accStakeKey(actorType, accName string) string {
	return fmt.Sprintf("stake/%s/%s", actorType, accName)
}
