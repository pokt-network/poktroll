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
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	sdklog "cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cometcli "github.com/cometbft/cometbft/libs/cli"
	cometjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/gorilla/websocket"
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
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	numQueryRetries = uint8(3)
	unbondingPeriod = "unbonding"
	transferPeriod  = "transfer"
)

var (
	addrRe          *regexp.Regexp
	amountRe        *regexp.Regexp
	addrAndAmountRe *regexp.Regexp

	accNameToAddrMap             = make(map[string]string)
	accAddrToNameMap             = make(map[string]string)
	accNameToAppMap              = make(map[string]apptypes.Application)
	operatorAccNameToSupplierMap = make(map[string]sharedtypes.Supplier)

	flagFeaturesPath string
	keyRingFlag      = "--keyring-backend=test"
	chainIdFlag      = "--chain-id=poktroll"
	// pathUrl points to a local gateway using the PATH framework in centralized mode.
	pathUrl = "http://localhost:3000/v1" // localhost is kept as the default to streamline local development & testing.
)

func init() {
	addrRe = regexp.MustCompile(`address:\s+(\S+)\s+name:\s+(\S+)`)
	amountRe = regexp.MustCompile(`amount:\s+"(.+?)"\s+denom:\s+upokt`)
	addrAndAmountRe = regexp.MustCompile(`(?s)address: ([\w\d]+).*?stake:\s*amount: "(\d+)"`)

	flag.StringVar(&flagFeaturesPath, "features-path", "*.feature", "Specifies glob paths for the runner to look up .feature files")

	// If "PATH_URL" ENV variable is present, use it for pathUrl
	if url := os.Getenv("PATH_URL"); url != "" {
		pathUrl = url
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.Printf("Running features matching %q", path.Join("e2e", "tests", flagFeaturesPath))
	m.Run()
}

// evmSubscription is a struct to unmarshal the JSON response from the EVM subscription.
type evmSubscription struct {
	Method string `json:"method"`
	Params struct {
		Result struct {
			Hash   string `json:"hash"`
			Number string `json:"number"`
		} `json:"result"`
	} `json:"params"`
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

	// wsConn is the websocket connection to the PATH (i.e. gateway) websockets endpoint.
	wsConn *websocket.Conn
	// numEVMSubscriptionEvents is the number of eth subscription events received
	// from the RelayMiner through the Gateway's websocket connection.
	numEVMSubscriptionEvents atomic.Uint64
	// wsCloseHeight is the block height at which the websocket connection should be closed
	// by the relay miner due to the end of the session.
	wsCloseHeight int64

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
	gocuke.NewRunner(t, &suite{}).Path(flagFeaturesPath).
		// Ignore test elements (e.g. Features, Scenarios, etc.)
		// with the @manual tag (e.g. migration.feature).
		Tags("not @manual").Run()
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
	require.NoError(s, err, "error running command %s due to: %v", cmd, err)
	s.pocketd.result = res
}

func (s *suite) TheUserShouldBeAbleToSeeStandardOutputContaining(arg1 string) {
	require.Containsf(s, s.pocketd.result.Stdout, arg1, s.pocketd.result.Stderr)
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
	require.NoError(s, err, "error sending upokt from %q to %q due to: %v", accName1, accName2, err)
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

func (s *suite) TheAccountBalanceOfShouldBeUpoktThanBefore(accName string, expectedBalanceChange int64, condition string) {
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
	s.validateAmountChange(prevBalance, currBalance, expectedBalanceChange, accName, condition, "balance")
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
	require.NoError(s, err, "error staking %s due to: %v", actorType, err)

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
	accAddress := accNameToAddrMap[accName]
	configContent := s.getConfigFileContent(amount, accAddress, accAddress, actorType, serviceId)
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
	require.NoError(s, err, "error staking %s for service %s due to: %v", actorType, serviceId, err)

	// Remove the temporary config file
	err = os.Remove(configFile.Name())
	require.NoError(s, err, "error removing config file %q", configFile.Name())

	s.pocketd.result = res
}

func (s *suite) getConfigFileContent(
	amount int64,
	ownerAddress,
	operatorAddress,
	actorType,
	serviceId string,
) string {
	var configContent string
	switch actorType {
	case apptypes.ModuleName:
		configContent = fmt.Sprintf(`
		stake_amount: %dupokt
		service_ids:
		  - %s`,
			amount, serviceId)
	case suppliertypes.ModuleName:
		configContent = fmt.Sprintf(`
			owner_address: %s
			operator_address: %s
			stake_amount: %dupokt
			services:
			  - service_id: %s
			    endpoints:
			    - publicly_exposed_url: http://relayminer1:8545
			      rpc_type: json_rpc`,
			ownerAddress, operatorAddress, amount, serviceId)
	default:
		s.Fatalf("ERROR: unknown actor type %s", actorType)
	}
	return yaml.NormalizeYAMLIndentation(configContent)
}

func (s *suite) TheUserUnstakesAFromTheAccount(actorType string, accName string) {
	var args []string

	switch actorType {
	case suppliertypes.ModuleName:
		args = []string{
			"tx",
			actorType,
			fmt.Sprintf("unstake-%s", actorType),
			accNameToAddrMap[accName], // supplier owner or operator address
			"--from",
			accName,
			keyRingFlag,
			chainIdFlag,
			"-y",
		}
	default:
		args = []string{
			"tx",
			actorType,
			fmt.Sprintf("unstake-%s", actorType),
			"--from",
			accName,
			keyRingFlag,
			chainIdFlag,
			"-y",
		}
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "error unstaking %s due to: %v", actorType, err)

	// Get current balance
	balanceKey := accBalanceKey(accName)
	currBalance := s.getAccBalance(accName)
	s.scenarioState[balanceKey] = currBalance // save the balance for later

	// NB: s.pocketd.result MUST be set AFTER the balance is queried because the
	// balance query sets the result first while getting the account balance.
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

	// CHeck if the application is registered for the service
	isRegistered := false
	for _, serviceConfig := range app.ServiceConfigs {
		if serviceConfig.ServiceId == serviceId {
			isRegistered = true
			break
		}
	}
	require.True(s, isRegistered, "application %s is not registered for service %s", appName, serviceId)

	cuprActual := s.getServiceComputeUnitsPerRelay(serviceId)
	cuprExpected, err := strconv.ParseUint(cuprStr, 10, 64)
	require.NoError(s, err)
	require.Equal(s, cuprExpected, cuprActual, "compute units per relay for service %s is not %d", serviceId, cuprExpected)
}

func (s *suite) TheUserVerifiesTheForAccountIsNotStaked(actorType, accName string) {
	_, ok := s.getStakedAmount(actorType, accName)
	require.Falsef(s, ok, "account %s of type %s SHOULD NOT be staked", accName, actorType)
}

func (s *suite) TheForAccountIsStakedWithUpokt(actorType, accName string, amount int64) {
	stakeAmount, ok := s.getStakedAmount(actorType, accName)
	require.Truef(s, ok, "account %s of type %s SHOULD be staked", accName, actorType)
	require.Equalf(s, amount, int64(stakeAmount), "account %s stake amount is not %d", accName, amount)
	s.scenarioState[accStakeKey(actorType, accName)] = stakeAmount // save the stakeAmount for later
}

func (s *suite) TheApplicationIsStakedForService(appName string, serviceId string) {
	for _, serviceConfig := range accNameToAppMap[appName].ServiceConfigs {
		if serviceConfig.ServiceId == serviceId {
			return
		}
	}
	s.Fatalf("ERROR: application %s is not staked for service %s", appName, serviceId)
}

func (s *suite) TheSupplierIsStakedForService(supplierOperatorName string, serviceId string) {
	for _, serviceConfig := range operatorAccNameToSupplierMap[supplierOperatorName].Services {
		if serviceConfig.ServiceId == serviceId {
			return
		}
	}
	s.Fatalf("ERROR: supplier %s is not staked for service %s", supplierOperatorName, serviceId)
}

func (s *suite) TheSessionForApplicationAndServiceContainsTheSupplier(appName string, serviceId string, supplierOperatorName string) {
	expectedSupplier, ok := operatorAccNameToSupplierMap[supplierOperatorName]
	require.True(s, ok, "supplier %s not found", supplierOperatorName)

	session := s.getSession(appName, serviceId)
	for _, supplier := range session.Suppliers {
		if supplier.OperatorAddress == expectedSupplier.OperatorAddress {
			return
		}
	}
	s.Fatalf("ERROR: session for app %s and service %s does not contain supplier %s", appName, serviceId, supplierOperatorName)
}

func (s *suite) TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPathAndData(appName, supplierOperatorName, serviceId, path, requestData string) {
	method := "POST"
	// If requestData is empty, assume a GET request
	if requestData == "" {
		method = "GET"
	}

	appAddr := accNameToAddrMap[appName]

	res, err := s.pocketd.RunCurlWithRetry(pathUrl, serviceId, method, path, appAddr, requestData, 5)
	require.NoError(s, err, "error sending relay request from app %q to supplier %q for service %q due to: %v", appName, supplierOperatorName, serviceId, err)

	var jsonContent json.RawMessage
	err = json.Unmarshal([]byte(res.Stdout), &jsonContent)
	require.NoErrorf(s, err, `Expected valid JSON, got: %s`, res.Stdout)

	jsonMap, err := jsonToMap(jsonContent)
	require.NoError(s, err, "error converting JSON to map")

	// Log the JSON content if the test is verbose
	if isVerbose() {
		prettyJson, err := jsonPrettyPrint(jsonContent)
		require.NoError(s, err, "error pretty printing JSON")
		s.Log(prettyJson)
	}

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

func (s *suite) TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPath(appName, supplierName, serviceId, path string) {
	s.TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPathAndData(appName, supplierName, serviceId, path, "")
}

func (s *suite) AModuleEndBlockEventIsBroadcast(module, eventType string) {
	s.waitForNewBlockEvent(newEventTypeMatchFn(module, eventType))
}

func (s *suite) TheSupplierForAccountIsUnbonding(supplierOperatorName string) {
	_, ok := operatorAccNameToSupplierMap[supplierOperatorName]
	require.True(s, ok, "supplier %s not found", supplierOperatorName)

	s.waitForTxResultEvent(newEventMsgTypeMatchFn("supplier", "UnstakeSupplier"))

	supplier := s.getSupplierInfo(supplierOperatorName)
	require.True(s, supplier.IsUnbonding())
}

func (s *suite) TheGatewayForAccountIsUnbonding(gatewayName string) {
	_, ok := accNameToAddrMap[gatewayName]
	require.True(s, ok, "gateway %s not found", gatewayName)

	s.waitForTxResultEvent(newEventMsgTypeMatchFn("gateway", "UnstakeGateway"))

	gateway := s.getGatewayInfo(gatewayName)
	require.True(s, gateway.IsUnbonding())
}

func (s *suite) TheUserWaitsForTheSupplierForAccountUnbondingPeriodToFinish(accName string) {
	_, ok := operatorAccNameToSupplierMap[accName]
	require.True(s, ok, "supplier %s not found", accName)

	unbondingHeight := s.getSupplierUnbondingEndHeight(accName)
	s.waitForBlockHeight(unbondingHeight + 1) // Add 1 to ensure the unbonding block has been committed
}

func (s *suite) TheApplicationForAccountIsInThePeriod(appName, periodName string) {
	_, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

	var (
		msgType      string
		isAppInState func(*apptypes.Application) bool
	)
	switch periodName {
	case unbondingPeriod:
		msgType = "UnstakeApplication"
		isAppInState = func(app *apptypes.Application) bool {
			return app.IsUnbonding()
		}
	case transferPeriod:
		msgType = "TransferApplication"
		isAppInState = func(application *apptypes.Application) bool {
			return application.HasPendingTransfer()
		}
	default:
		s.Fatalf("unsupported period type: %q", periodName)
	}

	s.waitForTxResultEvent(newEventMsgTypeMatchFn("application", msgType))

	application := s.getApplicationInfo(appName)
	require.True(s, isAppInState(application))
}

func (s *suite) TheUserWaitsForTheApplicationForAccountPeriodToFinish(accName, periodType string) {
	_, ok := accNameToAppMap[accName]
	require.True(s, ok, "application %s not found", accName)

	// TODO_IMPROVE: Add an event to listen for instead. This will require
	// refactoring and/or splitting of this method for each event type.

	switch periodType {
	case unbondingPeriod:
		unbondingHeight := s.getApplicationUnbondingHeight(accName)
		s.waitForBlockHeight(unbondingHeight + 1) // Add 1 to ensure the unbonding block has been committed
	case transferPeriod:
		transferEndHeight := s.getApplicationTransferEndHeight(accName)
		s.waitForBlockHeight(transferEndHeight + 1) // Add 1 to ensure the transfer end block has been committed
	}

	// Rebuild app map after the relevant period has elapsed.
	s.buildAppMap()
}

func (s *suite) getStakedAmount(actorType, accName string) (int, bool) {
	s.Helper()

	listCommand := fmt.Sprintf("list-%s", actorType)
	// TODO_TECHDEBT(@olshansky): As of #1028, we started migrating some parts
	// of the CLI to use AutoCLI which made list commands pluralized.
	// E.g. "list-suppliers" instead of "list-supplier".
	// Over time, all actor commands will be updated like so and this if can
	// be removed.
	if actorType == suppliertypes.ModuleName {
		listCommand = fmt.Sprintf("%ss", listCommand)
	}
	args := []string{
		"query",
		actorType,
		listCommand,
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

func (s *suite) TheUserShouldSeeThatTheSupplierForAccountIsStaked(supplierOperatorName string) {
	supplier := s.getSupplierInfo(supplierOperatorName)
	operatorAccNameToSupplierMap[accAddrToNameMap[supplier.OperatorAddress]] = *supplier
	require.NotNil(s, supplier, "supplier %s not found", supplierOperatorName)
}

func (s *suite) TheSessionForApplicationAndServiceDoesNotContain(appName, serviceId, supplierOperatorName string) {
	session := s.getSession(appName, serviceId)

	for _, supplier := range session.Suppliers {
		if supplier.OperatorAddress == accNameToAddrMap[supplierOperatorName] {
			s.Fatalf(
				"ERROR: session for app %s and service %s should not contain supplier %s",
				appName,
				serviceId,
				supplierOperatorName,
			)
		}
	}
}

func (s *suite) TheUserWaitsForSupplierToBecomeActiveForService(supplierOperatorName, serviceId string) {
	supplier := s.getSupplierInfo(supplierOperatorName)
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
		"list-suppliers",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoError(s, err, "error getting supplier list")
	s.pocketd.result = res
	var resp suppliertypes.QueryAllSuppliersResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	for _, supplier := range resp.Supplier {
		operatorAccNameToSupplierMap[accAddrToNameMap[supplier.OperatorAddress]] = supplier
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

// getSupplierInfo returns the supplier information for a given supplier operator address
func (s *suite) getSupplierInfo(supplierOperatorName string) *sharedtypes.Supplier {
	supplierOperatorAddr := accNameToAddrMap[supplierOperatorName]
	args := []string{
		"query",
		"supplier",
		"show-supplier",
		supplierOperatorAddr,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting supplier %s due to error: %v", supplierOperatorAddr, err)
	s.pocketd.result = res

	var resp suppliertypes.QueryGetSupplierResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	return &resp.Supplier
}

// getGatewayInfo returns the gateway information for a given gateway account address
func (s *suite) getGatewayInfo(gatewayName string) *gatewaytypes.Gateway {
	gatewayAddr := accNameToAddrMap[gatewayName]
	args := []string{
		"query",
		"gateway",
		"show-gateway",
		gatewayAddr,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting gateway %s due to error: %v", gatewayAddr, err)
	s.pocketd.result = res

	var resp gatewaytypes.QueryGetGatewayResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	return &resp.Gateway
}

// getSupplierUnbondingEndHeight returns the height at which the supplier will be unbonded.
func (s *suite) getSupplierUnbondingEndHeight(accName string) int64 {
	supplier := s.getSupplierInfo(accName)
	sharedParams := s.getSharedParams()

	return sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, supplier)
}

// getApplicationInfo returns the application information for a given application address.
func (s *suite) getApplicationInfo(appName string) *apptypes.Application {
	appAddr := accNameToAddrMap[appName]
	args := []string{
		"query",
		"application",
		"show-application",
		appAddr,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting supplier %s", appAddr)
	s.pocketd.result = res

	var resp apptypes.QueryGetApplicationResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	return &resp.Application
}

// getApplicationUnbondingHeight returns the height at which the application will be unbonded.
func (s *suite) getApplicationUnbondingHeight(accName string) int64 {
	application := s.getApplicationInfo(accName)

	sharedParams := s.getSharedParams()
	unbondingHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, application)
	return unbondingHeight
}

// getApplicationTransferEndHeight returns the height at which the application will be transferred to the destination.
func (s *suite) getApplicationTransferEndHeight(accName string) int64 {
	application := s.getApplicationInfo(accName)
	require.NotNil(s, application.GetPendingTransfer())

	sharedParams := s.getSharedParams()

	return apptypes.GetApplicationTransferHeight(&sharedParams, application)
}

// getServiceComputeUnitsPerRelay returns the compute units per relay for a given service ID
func (s *suite) getServiceComputeUnitsPerRelay(serviceId string) uint64 {
	args := []string{
		"query",
		"service",
		"show-service",
		serviceId,
		"--output=json",
	}

	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoError(s, err, "error getting shared module params")

	var resp servicetypes.QueryGetServiceResponse
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	s.cdc.MustUnmarshalJSON(responseBz, &resp)
	return resp.Service.ComputeUnitsPerRelay
}

// getSharedParams returns the shared module parameters
func (s *suite) getSharedParams() sharedtypes.Params {
	args := []string{
		"query",
		"shared",
		"params",
		"--output=json",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "error querying shared params")

	var sharedParamsRes sharedtypes.QueryParamsResponse

	s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &sharedParamsRes)
	require.NoError(s, err)

	return sharedParamsRes.Params
}

// getCurrentBlockHeight returns the current block height
func (s *suite) getCurrentBlockHeight() int64 {
	args := []string{
		"query",
		"block",
		"--output=json",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err, "error querying for the latest block")

	// Remove the first line of the response to avoid unmarshalling non JSON data.
	// This is needed because, when no height is provided, the query block command returns:
	//   "Falling back to latest block height:"
	stdoutLines := strings.Split(res.Stdout, "\n")
	require.Greater(s, len(stdoutLines), 1, "expected at least one line of output")
	res.Stdout = strings.Join(stdoutLines[1:], "\n")

	var blockRes struct {
		LastCommit struct {
			Height int64 `json:"height"`
		} `json:"last_commit"`
	}

	err = cometjson.Unmarshal([]byte(res.Stdout), &blockRes)
	require.NoError(s, err)

	return blockRes.LastCommit.Height
}

// readEVMSubscriptionEvents reads the eth_subscription events from the websocket
// connection until it gets closed and increments the number of events received.
func (s *suite) readEVMSubscriptionEvents() context.Context {
	s.numEVMSubscriptionEvents.Store(0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			_, message, err := s.wsConn.ReadMessage()
			// Read messages until the connection is closed.
			if err != nil {
				cancel()
				return
			}

			var evmSubscriptionMsg evmSubscription
			if err = json.Unmarshal(message, &evmSubscriptionMsg); err != nil {
				continue
			}

			if evmSubscriptionMsg.Method != "eth_subscription" {
				continue
			}

			// Ensure the hash and number are populated.
			require.True(s, strings.HasPrefix(evmSubscriptionMsg.Params.Result.Hash, "0x"))
			require.True(s, strings.HasPrefix(evmSubscriptionMsg.Params.Result.Number, "0x"))
			s.numEVMSubscriptionEvents.Add(1)
		}
	}()

	return ctx
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
