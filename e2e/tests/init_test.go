//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
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
	abci "github.com/cometbft/cometbft/abci/types"
	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/yaml"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	shared "github.com/pokt-network/poktroll/x/shared"
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
	pocketd *pocketdBin

	// TODO_IMPROVE: refactor all usages of scenarioState to be fields on the suite struct.
	scenarioState    map[string]any // temporary state for each scenario
	cdc              codec.Codec
	proofQueryClient prooftypes.QueryClient

	// See the Cosmo SDK authz module for references related to `granter` and `grantee`
	// https://docs.cosmos.network/main/build/modules/authz
	granterName string
	granteeName string

	// moduleParamsMap is a map of module names to a map of parameter names to parameter values & types.
	expectedModuleParams moduleParamsMap

	deps                       depinject.Config
	newBlockEventsReplayClient client.EventsReplayClient[*block.CometNewBlockEvent]
	txResultReplayClient       client.EventsReplayClient[*abci.TxResult]
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

	s.deps = depinject.Supply(
		events.NewEventsQueryClient(testclient.CometLocalWebsocketURL),
	)

	// Start the NewBlockEventsReplayClient before the test so that it can't miss any block events.
	s.newBlockEventsReplayClient, err = events.NewEventsReplayClient[*block.CometNewBlockEvent](
		s.ctx,
		s.deps,
		"tm.event='NewBlock'",
		block.UnmarshalNewBlockEvent,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)

	s.txResultReplayClient, err = events.NewEventsReplayClient[*abci.TxResult](
		s.ctx,
		s.deps,
		"tm.event='Tx'",
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
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
	require.NoError(s, err, "error staking %s for service %s", actorType, serviceId)

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
		s.Fatalf("ERROR: unknown actor type %s", actorType)
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

func (s *suite) TheServiceRegisteredForApplicationHasAComputeUnitsPerRelayOf(serviceId string, appName string, cuprStr string) {
	app, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

	for _, serviceConfig := range app.ServiceConfigs {
		if serviceConfig.Service.Id == serviceId {
			cupr, err := strconv.ParseUint(cuprStr, 10, 64)
			require.NoError(s, err)
			require.Equal(s, cupr, serviceConfig.Service.ComputeUnitsPerRelay)
			return
		}
	}
	s.Fatalf("ERROR: service %s is not registered for application %s", serviceId, appName)
}

func (s *suite) TheUserVerifiesTheForAccountIsNotStaked(actorType, accName string) {
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
	s.Fatalf("ERROR: application %s is not staked for service %s", appName, serviceId)
}

func (s *suite) TheSupplierIsStakedForService(supplierName string, serviceId string) {
	for _, serviceConfig := range accNameToSupplierMap[supplierName].Services {
		if serviceConfig.Service.Id == serviceId {
			return
		}
	}
	s.Fatalf("ERROR: supplier %s is not staked for service %s", supplierName, serviceId)
}

func (s *suite) TheSessionForApplicationAndServiceContainsTheSupplier(appName string, serviceId string, supplierName string) {
	expectedSupplier, ok := accNameToSupplierMap[supplierName]
	require.True(s, ok, "supplier %s not found", supplierName)

	session := s.getSession(appName, serviceId)
	for _, supplier := range session.Suppliers {
		if supplier.Address == expectedSupplier.Address {
			return
		}
	}
	s.Fatalf("ERROR: session for app %s and service %s does not contain supplier %s", appName, serviceId, supplierName)
}

func (s *suite) TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPathAndData(appName, supplierName, serviceId, path, requestData string) {
	// TODO_HACK: We need to support a non self_signing LocalNet AppGateServer
	// that allows any application to send a relay in LocalNet and our E2E Tests.
	require.Equal(s, "app1", appName, "TODO_HACK: The LocalNet AppGateServer is self_signing and only supports app1.")

	method := "POST"
	// If requestData is empty, assume a GET request
	if requestData == "" {
		method = "GET"
	}

	res, err := s.pocketd.RunCurlWithRetry(appGateServerUrl, serviceId, method, path, requestData, 5)
	require.NoError(s, err, "error sending relay request from app %q to supplier %q for service %q", appName, supplierName, serviceId)

	var jsonContent json.RawMessage
	err = json.Unmarshal([]byte(res.Stdout), &jsonContent)
	require.NoError(s, err, `Expected valid JSON, got: %s`, res.Stdout)

	jsonMap, err := jsonToMap(jsonContent)
	require.NoError(s, err, "error converting JSON to map")

	prettyJson, err := jsonPrettyPrint(jsonContent)
	require.NoError(s, err, "error pretty printing JSON")
	s.Log(prettyJson)

	// TODO_IMPROVE: This is a minimalistic first approach to request validation in E2E tests.
	// Consider leveraging the shannon-sdk or path here.
	switch path {
	case "":
		// Validate JSON-RPC request where the path is empty
		require.Nil(s, jsonMap["error"], "error in relay response")
		require.NotNil(s, jsonMap["result"], "no result in relay response")
	default:
		// Validate REST request where the path is non-empty
		require.Nil(s, jsonMap["error"], "error in relay response")
	}
}

func (s *suite) TheApplicationSendsTheSupplierARequestForServiceWithPath(appName, supplierName, serviceId, path string) {
	s.TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPathAndData(appName, supplierName, serviceId, path, "")
}

func (s *suite) AModuleEndBlockEventIsBroadcast(module, eventType string) {
	s.waitForNewBlockEvent(newEventTypeMatchFn(module, eventType))
}

func (s *suite) TheSupplierForAccountIsUnbonding(accName string) {
	_, ok := accNameToSupplierMap[accName]
	require.True(s, ok, "supplier %s not found", accName)

	s.waitForTxResultEvent(newEventMsgTypeMatchFn("supplier", "UnstakeSupplier"))

	supplier := s.getSupplierInfo(accName)
	require.True(s, supplier.IsUnbonding())
}

func (s *suite) TheUserWaitsForUnbondingPeriodToFinish(accName string) {
	_, ok := accNameToSupplierMap[accName]
	require.True(s, ok, "supplier %s not found", accName)

	unbondingHeight := s.getSupplierUnbondingHeight(accName)
	s.waitForBlockHeight(unbondingHeight + 1) // Add 1 to ensure the unbonding block has been committed
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

func (s *suite) TheUserShouldSeeThatTheSupplierForAccountIsStaked(supplierName string) {
	supplier := s.getSupplierInfo(supplierName)
	accNameToSupplierMap[accAddrToNameMap[supplier.Address]] = *supplier
	require.NotNil(s, supplier, "supplier %s not found", supplierName)
}

func (s *suite) TheSessionForApplicationAndServiceDoesNotContain(appName, serviceId, supplierName string) {
	session := s.getSession(appName, serviceId)

	for _, supplier := range session.Suppliers {
		if supplier.Address == accNameToAddrMap[supplierName] {
			s.Fatalf(
				"ERROR: session for app %s and service %s should not contain supplier %s",
				appName,
				serviceId,
				supplierName,
			)
		}
	}
}

func (s *suite) TheUserWaitsForSupplierToBecomeActiveForService(supplierName, serviceId string) {
	supplier := s.getSupplierInfo(supplierName)
	s.waitForBlockHeight(int64(supplier.ServicesActivationHeightsMap[serviceId]))
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

// getSession returns the current session for the given application and service.
func (s *suite) getSession(appName string, serviceId string) *sessiontypes.Session {
	app, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

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

	return resp.Session
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
		s.Fatalf("ERROR: unknown condition %s", condition)
	}

}

// getSupplierInfo returns the supplier information for a given supplier address
func (s *suite) getSupplierInfo(supplierName string) *sharedtypes.Supplier {
	supplierAddr := accNameToAddrMap[supplierName]
	args := []string{
		"query",
		"supplier",
		"show-supplier",
		supplierAddr,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting supplier %s", supplierAddr)
	s.pocketd.result = res

	var resp suppliertypes.QueryGetSupplierResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	return &resp.Supplier
}

// getSupplierUnbondingHeight returns the height at which the supplier will be unbonded.
func (s *suite) getSupplierUnbondingHeight(accName string) int64 {
	supplier := s.getSupplierInfo(accName)

	args := []string{
		"query",
		"shared",
		"params",
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting shared module params")

	var resp sharedtypes.QueryParamsResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	unbondingHeight := shared.GetSupplierUnbondingHeight(&resp.Params, supplier)
	return unbondingHeight
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

// printPrettyJSON returns the given raw JSON message in a pretty format that
// can be printed to the console.
func jsonPrettyPrint(raw json.RawMessage) (string, error) {
	var buf bytes.Buffer
	err := json.Indent(&buf, raw, "", "  ")
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// jsonToMap converts a raw JSON message into a map of string keys and interface values.
func jsonToMap(raw json.RawMessage) (map[string]interface{}, error) {
	var dataMap map[string]interface{}

	// Unmarshal the raw message into the map
	err := json.Unmarshal(raw, &dataMap)
	if err != nil {
		return nil, err
	}

	return dataMap, nil
}
