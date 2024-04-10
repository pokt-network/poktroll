package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/api/poktroll/tokenomics"
	"github.com/pokt-network/poktroll/cmd/signals"
	config "github.com/pokt-network/poktroll/load-testing/config"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/sync2"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	// maxConcurrentRequestLimit is the maximum number of concurrent requests that can be made.
	// By default, it is set to the number of logical CPUs available to the process.
	maxConcurrentRequestLimit = runtime.GOMAXPROCS(0)
	// fundingAccountKeyName is the key name of the account used to fund other accounts.
	fundingAccountKeyName = "pnf"
	// fundingAmount is the amount of tokens to fund other accounts.
	fundingAmount = sdk.NewCoin("upokt", math.NewInt(10000000))
	// stakeAmount is the amount of tokens to stake for suppliers and gateways.
	stakeAmount = sdk.NewCoin("upokt", math.NewInt(2000))
	// usedService is the service ID for the Anvil service that all applications
	// and suppliers will be using in this test.
	usedService = &sharedtypes.Service{Id: "anvil"}
	// loadTestManifestPath is the path to the load test manifest file.
	// It is used to the provisioned gateways and suppliers to use in the test.
	// TODO_BLOCKER: Get the path of the load test manifest from CLI flags.
	loadTestManifestPath = "../../loadtest_manifest.yaml"
	// blockDuration is the duration of a block in seconds.
	blockDuration = int64(1)
)

// relaysSuite is a test suite for the relays stress test.
// It tests the performance of the relays module by sending a large number of relay requests
// concurrently to a network of applications, gateways, and suppliers.
// The test is parameterized by the number of applications, gateways, and suppliers to be staked,
// and the rate at which new actors are staked.
type relaysSuite struct {
	gocuke.TestingT
	// ctx is the global context for the test suite.
	// It is cancelled when the test suite is cleaned up causing all goroutines
	// and observables subscriptions to be cancelled.
	ctx context.Context
	// cancelCtx is the cancel function for the global context.
	cancelCtx context.CancelFunc

	// blockClient is the block client used to subscribe to committed blocks and
	// query for the last block height.
	blockClient client.BlockClient
	// blocksReplayObs is the observable for committed blocks.
	// It is created from the block client and used to notify the test suite
	// of new blocks committed.
	blocksReplayObs client.BlockReplayObservable
	// sessionInfoObs is the observable mapping committed blocks to session information.
	// It is used to determine when to stake new actors and when they become active.
	sessionInfoObs observable.Observable[*sessionInfoNotif]

	// txContext is the transaction context used to sign and send transactions.
	txContext client.TxContext

	// fundingAccountInfo is the account entry corresponding to the fundingAccountKeyName.
	fundingAccountInfo *accountInfo
	// relaysSent is the number of relay requests sent.
	relaysSent atomic.Uint64
	// relayRatePerApp is the rate of relay requests per application per second.
	relayRatePerApp int64
	// relayCost is the cost of a relay request.
	relayCost int64

	// waitingForFirstSession is a flag indicating whether the test is waiting for
	// the first session to start.
	waitingForFirstSession bool
	// startBlockHeight is the block height at which the test started.
	startBlockHeight int64
	// testDurationBlocks is the duration of the test in blocks.
	testDurationBlocks int64

	// gatewayInitialCount is the number of gateways available at the start of the test.
	gatewayInitialCount int64
	// supplierInitialCount is the number of suppliers available at the start of the test.
	supplierInitialCount int64
	// appInitialCount is the number of applications available at the start of the test.
	appInitialCount int64

	// provisionedGateways is the list of provisioned gateways.
	// These are the gateways that are available to be staked.
	// Since AppGateServers are pre-provisioned, and already assigned a signingKeyName
	// and exposedServerAddress, the test suite does not create new ones but picks
	// from this list.
	provisionedGateways []*provisionedActorInfo
	// provisionedSuppliers is the list of provisioned suppliers.
	// These are the suppliers that are available to be staked.
	// Since RelayMiners are pre-provisioned, and already assigned a signingKeyName
	// and exposedServerAddress, the test suite does not create new ones but picks
	// from this list and use its information to create StakeSupplierMsgs
	provisionedSuppliers []*provisionedActorInfo

	// preparedGateways is the list of gateways that are already staked and ready
	// to be used in the next session.
	// They are segregated from activeGateways to avoid sending relay requests
	// to them since the delegation will be active in the next session.
	preparedGateways []*provisionedActorInfo
	// preparedApplications is the list of applications that are already staked and ready
	// to be used in the next session.
	// They are segregated from activeApplications to avoid sending relay requests
	// from them since their delegations will be active in the next session.
	preparedApplications []*applicationInfo
	// preparedSuppliers is the list of suppliers that are already staked and ready
	// to be used in the next session.
	preparedSuppliers []*provisionedActorInfo

	// activeGateways is the list of gateways that are currently staked and active.
	// They are used to send relay requests to the staked suppliers.
	activeGateways []*provisionedActorInfo
	// activeApplications is the list of applications that are currently staked and
	// used to send relays to the gateways they delegated to.
	activeApplications []*applicationInfo
	// activeSuppliers is the list of suppliers that are currently staked and
	// ready to handle relay requests.
	activeSuppliers []*provisionedActorInfo
}

// accountInfo contains the account info needed to build and send transactions.
type accountInfo struct {
	keyName     string
	accAddress  sdk.AccAddress
	pendingMsgs []sdk.Msg
}

// provisionedActorInfo represents gateways and suppliers that are provisioned
// with their respective keyName and exposedUrl.
// The supplier exposedUrl is the address advertised when staking a supplier.
// The gateway exposedUrl is the address used to send relay requests to the gateway.
type provisionedActorInfo struct {
	accountInfo
	exposedUrl string
}

// applicationInfo represents the dynamically created applications with their
// corresponding private keys and amount to stake which is calculated based on
// the time the application was created.
type applicationInfo struct {
	accountInfo
	amountToStake sdk.Coin
	privKey       *secp256k1.PrivKey
}

// sessionInfoNotif is a struct containing the session information of a block.
type sessionInfoNotif struct {
	blockHeight             int64
	sessionNumber           int64
	sessionStartBlockHeight int64
	sessionEndBlockHeight   int64
}

func TestLoadRelays(t *testing.T) {
	gocuke.NewRunner(t, &relaysSuite{}).Path(filepath.Join(".", "relays_stress.feature")).Run()
}

func (s *relaysSuite) LocalnetIsRunning() {
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	// Cancel the context if this process is interrupted or exits.
	// Delete the keyring entries for the application accounts since they are
	// not persisted across test runs.
	signals.GoOnExitSignal(func() {
		fmt.Println("")
		for _, app := range s.activeApplications {
			s.txContext.GetKeyring().Delete(app.keyName)
		}
		for _, app := range s.preparedApplications {
			s.txContext.GetKeyring().Delete(app.keyName)
		}
		s.cancelCtx()
	})

	s.Cleanup(func() {
		for _, app := range s.activeApplications {
			s.txContext.GetKeyring().Delete(app.keyName)
		}
		for _, app := range s.preparedApplications {
			s.txContext.GetKeyring().Delete(app.keyName)
		}
	})

	// Set up the block client.
	s.blockClient = testblock.NewLocalnetClient(s.ctx, s.TestingT.(*testing.T))

	// Setup the txClient
	s.txContext = testtx.NewLocalnetContext(s.TestingT.(*testing.T))

	// Set up the observable that will be notifying the suite about the committed blocks.
	s.blocksReplayObs = s.blockClient.CommittedBlocksSequence(s.ctx)

	// Set up the tokenomics client.
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	tokenomicsClient := tokenomics.NewQueryClient(clientCtx)

	// Get the relay cost from the tokenomics module.
	res, err := tokenomicsClient.Params(s.ctx, &tokenomics.QueryParamsRequest{})
	require.NoError(s, err)

	s.relayCost = int64(res.Params.ComputeUnitsToTokensMultiplier)

	// Initialize the funding account.
	s.initFundingAccount(fundingAccountKeyName)

	loadTestManifestContent, err := os.ReadFile(loadTestManifestPath)
	require.NoError(s, err)

	provisionedActors, err := config.ParseLoadTestManifest(loadTestManifestContent)
	require.NoError(s, err)

	for _, gateway := range provisionedActors.Gateways {
		s.provisionedGateways = append(s.provisionedGateways, &provisionedActorInfo{
			accountInfo: accountInfo{
				keyName: gateway.KeyName,
			},
			exposedUrl: gateway.ExposedUrl,
		})
	}

	for _, supplier := range provisionedActors.Suppliers {
		s.provisionedSuppliers = append(s.provisionedSuppliers, &provisionedActorInfo{
			accountInfo: accountInfo{
				keyName: supplier.KeyName,
			},
			exposedUrl: supplier.ExposedUrl,
		})
	}
}

func (s *relaysSuite) ARateOfRelayRequestsPerSecondIsSentPerApplication(appRPS string) {
	relayRatePerApp, err := strconv.ParseInt(appRPS, 10, 32)
	require.NoError(s, err)

	s.relayRatePerApp = relayRatePerApp
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	s.supplierInitialCount = table.Cell(3, 1).Int64()
	s.addInitialSuppliers(s.supplierInitialCount)

	s.gatewayInitialCount = table.Cell(1, 1).Int64()
	s.addInitialGateways(s.gatewayInitialCount)

	s.appInitialCount = table.Cell(2, 1).Int64()
	s.addInitialApplications(s.appInitialCount)
}

func (s *relaysSuite) MoreActorsAreStakedAsFollows(table gocuke.DataTable) {
	gatewayInc := table.Cell(1, 1).Int64()
	gatewayBlockIncRate := table.Cell(1, 2).Int64()
	require.Truef(s, gatewayBlockIncRate%keeper.NumBlocksPerSession == 0, "gateway increment rate must be a multiple of the session length")
	maxGateways := table.Cell(1, 3).Int64()
	require.Truef(s, len(s.provisionedGateways) >= int(maxGateways), "provisioned gateways must be greater or equal than the max gateways to be staked")

	supplierInc := table.Cell(3, 1).Int64()
	supplierBlockIncRate := table.Cell(3, 2).Int64()
	require.Truef(s, supplierBlockIncRate%keeper.NumBlocksPerSession == 0, "supplier increment rate must be a multiple of the session length")
	maxSuppliers := table.Cell(3, 3).Int64()
	require.Truef(s, len(s.provisionedSuppliers) >= int(maxSuppliers), "provisioned suppliers must be greater or equal than the max suppliers to be staked")

	appInc := table.Cell(2, 1).Int64()
	appBlockIncRate := table.Cell(2, 2).Int64()
	maxApps := table.Cell(2, 3).Int64()
	require.Truef(s, appBlockIncRate%keeper.NumBlocksPerSession == 0, "app increment rate must be a multiple of the session length")

	s.waitingForFirstSession = true

	// The test duration is the longest duration of the three actor increments.
	// The duration of each actor is calculated as how many blocks it takes to
	// increment the actor count to the maximum.
	s.testDurationBlocks = math.Max(
		maxGateways/gatewayInc*gatewayBlockIncRate,
		maxApps/appInc*appBlockIncRate,
		maxSuppliers/supplierInc*supplierBlockIncRate,
	)

	// Fund all the initial actors
	s.sendFundInitialActorsMsgs()
	s.waitForNextBlock()

	// Stake All the initial actors
	s.sendInitialActorsStakeMsgs()
	s.waitForNextBlock()

	// Delegate all the initial applications to the initial gateways
	s.sendInitialDelegateMsgs()
	s.waitForNextBlock()

	// sessionInfoObs maps committed blocks to a notification which includes the
	// session number and the start and end block heights of the session.
	// It runs at the same frequency as committed blocks (i.e. 1:1).
	s.sessionInfoObs = channel.Map(s.ctx, s.blocksReplayObs,
		func(
			ctx context.Context,
			block client.Block,
		) (*sessionInfoNotif, bool) {
			blockHeight := block.Height()
			sessionInfo := &sessionInfoNotif{
				blockHeight:             blockHeight,
				sessionNumber:           keeper.GetSessionNumber(blockHeight),
				sessionStartBlockHeight: keeper.GetSessionStartBlockHeight(blockHeight),
				sessionEndBlockHeight:   keeper.GetSessionEndBlockHeight(blockHeight),
			}

			sessionBlocksRemaining := sessionInfo.sessionEndBlockHeight - sessionInfo.blockHeight + 1

			// If the test has not started and the current block is not the first block
			// of the session, wait for the next session to start.
			if s.waitingForFirstSession && sessionInfo.blockHeight != sessionInfo.sessionStartBlockHeight {
				logger.Info().
					Int64("block_height", blockHeight).
					Int64("session_num", sessionInfo.sessionNumber).
					Msgf("waiting for next session to start: in %d blocks", sessionBlocksRemaining)

				return nil, true
			}

			// If the test has not started, set the start block height to the current block height.
			if s.waitingForFirstSession {
				s.startBlockHeight = blockHeight
			}

			// Mark the test as started.
			s.waitingForFirstSession = false

			logger.Info().
				Int64("block_heihgt", blockHeight).
				Msgf("progress: %d/%d", blockHeight-s.startBlockHeight, s.testDurationBlocks)
			if blockHeight >= s.startBlockHeight+s.testDurationBlocks {
				s.cancelCtx()
			}

			return sessionInfo, false
		},
	)

	// shouldBlockUpdateChainStateObs is an observable which is notified each block.
	// If the current "test height" is equal to the session start block height,
	// activate all the staked actors to be used in the current session.
	// If the current "test height" is a multiple of any actor increment block count,
	// stake new actors to be used in the next session.
	channel.ForEach(s.ctx, s.sessionInfoObs,
		func(ctx context.Context, notif *sessionInfoNotif) {
			// On the first block of each session, check if any new actors need to
			// be staked **for use in the next session**.
			// NB: assumes that the increment rates are multiples of the session length.
			// Otherwise, we would need to check if any block in the next session
			// is an increment height.

			s.activatePreparedActors(notif)

			if s.shouldIncrementActor(notif, supplierBlockIncRate, supplierInc, maxSuppliers) {
				// Incrementing suppliers can run concurrently with other actors since
				// they are not dependent on each other.
				go s.goIncrementSuppliers(notif, supplierInc, maxSuppliers)
			}

			// Get newly staked applications and gateways to create delegation messages
			// for them.
			// Stake messages are sent but not yet committed.

			var newGateways []*provisionedActorInfo
			if s.shouldIncrementActor(notif, gatewayBlockIncRate, gatewayInc, maxGateways) {
				newGateways = s.stakeGateways(notif, gatewayInc, maxGateways)
			}

			var newApps []*applicationInfo
			if s.shouldIncrementActor(notif, appBlockIncRate, appInc, maxApps) {
				newApps = s.fundApps(notif, appInc, maxApps)
			}

			if len(newApps) == 0 && len(newGateways) == 0 {
				return
			}

			// Wait for the next block to commit stake transactions and be able
			// to delegate the applications to the new gateways.
			s.waitForNextBlock()
			// After this call all applications are delegating to all the gateways.
			// This is to ensure that requests are distributed evenly across all gateways
			// at any given time.
			s.stakeAndDelegateApps(newApps, newGateways)
		},
	)
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentFromTheApplications() {
	// Limit the number of concurrent requests to maxConcurrentRequestLimit.
	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)
	for {
		select {
		// If the context is cancelled, the test is finished, stop sending relay requests.
		case <-s.ctx.Done():
			return
		default:
		}

		// Do not send relay requests if the test is waiting for the first session to start.
		// This is to align the relay requests with the active actors.
		if s.waitingForFirstSession {
			continue
		}

		// Calculate the realys per second as the number of active applications
		// each sending relayRatePerApp relays per second.
		relaysPerSec := len(s.activeApplications) * int(s.relayRatePerApp)
		// Determine the interval between each relay request.
		relayInterval := time.Second / time.Duration(relaysPerSec)

		batchLimiter.Go(s.ctx, func() {
			relaysSent := s.relaysSent.Add(1) - 1
			blockHeight := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
			sessionNum := keeper.GetSessionNumber(blockHeight)
			app := s.activeApplications[relaysSent%uint64(len(s.activeApplications))]
			gw := s.activeGateways[relaysSent%uint64(len(s.activeGateways))]

			logger.Info().
				Int64("session_num", sessionNum).
				Int64("block_height", blockHeight).
				Str("app", app.keyName).
				Str("gw", gw.keyName).
				Int("total_apps", len(s.activeApplications)).
				Int("total_gws", len(s.activeGateways)).
				Str("time", time.Now().Format(time.RFC3339Nano)).
				Msgf("sending relay #%d", relaysSent)
			s.sendRelay(relaysSent)
		})

		// Sleep for the interval between each relay request.
		// TODO_TECHDEBT: Cancel the sleep if the number of applications change,
		// so that the relay rate is adjusted accordingly, since sleeping while
		// the number of applications change will cause the relay rate to be slightly
		// off.
		time.Sleep(relayInterval)
	}
}

// stakeGateways stakes the next gatewayInc number of gateways, picks their keyName
// from the provisioned gateways list and sends the corresponding stake transactions.
func (s *relaysSuite) stakeGateways(
	sessionInfo *sessionInfoNotif,
	gatewayInc,
	maxGateways int64,
) (newGateways []*provisionedActorInfo) {
	gatewayCount := int64(len(s.activeGateways))

	gatewaysToStake := gatewayInc
	if gatewayCount+gatewaysToStake > maxGateways {
		gatewaysToStake = maxGateways - gatewayCount
	}

	if gatewaysToStake == 0 {
		return newGateways
	}

	logger.Debug().Msgf(
		"staking gateways for next session %d (%d->%d)",
		sessionInfo.sessionNumber+1,
		gatewayCount,
		gatewayCount+gatewaysToStake,
	)

	for gwIdx := int64(0); gwIdx < gatewaysToStake; gwIdx++ {
		gateway := s.addGateway(gatewayCount + gwIdx)
		s.generateStakeGatewayMsg(gateway)
		s.sendTx(&gateway.accountInfo)
		newGateways = append(newGateways, gateway)
	}

	// The new gateways are returned so the caller can construct delegation messages
	// given the existing applications.
	return newGateways
}

// fundApps creates the applications given the next appIncAmt and sends the corresponding
// fund transaction.
func (s *relaysSuite) fundApps(
	sessionInfo *sessionInfoNotif,
	appIncAmt,
	maxApps int64,
) (newApps []*applicationInfo) {
	appCount := int64(len(s.activeApplications))

	appsToStake := appIncAmt
	if appCount+appsToStake > maxApps {
		appsToStake = maxApps - appCount
	}

	if appsToStake == 0 {
		return newApps
	}

	logger.Debug().Msgf(
		"staking applications for next session %d (%d->%d)",
		sessionInfo.sessionNumber+1,
		appCount,
		appCount+appsToStake,
	)

	for appIdx := int64(0); appIdx < appsToStake; appIdx++ {
		app := s.createApplicationAccount(appCount + appIdx + 1)
		s.generateFundApplicationMsg(app, sessionInfo.sessionEndBlockHeight+1)
		newApps = append(newApps, app)
	}
	s.sendTx(s.fundingAccountInfo)

	// Then new applications are returned so the caller can construct delegation messages
	// given the existing gateways.
	return newApps
}

// stakeAndDelegateApps stakes the new applications and delegates them to both
// the active and new gateways.
// It also ensures that new gateways are delegated to the existing applications.
// It waits for the stake delegate messages to be committed before adding the new
// actors to their corresponding prepared lists.
func (s *relaysSuite) stakeAndDelegateApps(
	newApps []*applicationInfo,
	newGateways []*provisionedActorInfo,
) {
	for _, app := range s.activeApplications {
		for _, gateway := range newGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		s.sendTx(&app.accountInfo)
	}

	for _, app := range newApps {
		// Stake and delegate messages for a new application are sent in a single
		// transaction to avoid waiting for an additional block.
		s.generateStakeApplicationMsg(app)
		for _, gateway := range s.activeGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		for _, gateway := range newGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		s.sendTx(&app.accountInfo)
	}

	// Wait for the next block to commit the stake and delegate transactions.
	s.waitForNextBlock()
	s.preparedGateways = append(s.preparedGateways, newGateways...)
	s.preparedApplications = append(s.preparedApplications, newApps...)
}

// goIncrementSuppliers increments the number of suppliers to be staked.
// Staking new suppliers can run concurrently since it doesn't need to be
// synchronized with other actors.
func (s *relaysSuite) goIncrementSuppliers(
	sessionInfo *sessionInfoNotif,
	supplierInc,
	maxSuppliers int64,
) {
	supplierCount := int64(len(s.activeSuppliers))

	suppliersToStake := supplierInc
	if supplierCount+suppliersToStake > maxSuppliers {
		suppliersToStake = maxSuppliers - supplierCount
	}

	if suppliersToStake == 0 {
		return
	}

	logger.Debug().Msgf(
		"staking suppliers for next session %d (%d->%d)",
		sessionInfo.sessionNumber+1,
		supplierCount,
		supplierCount+suppliersToStake,
	)

	var newSuppliers []*provisionedActorInfo
	for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
		supplier := s.addSupplier(supplierCount + supplierIdx)
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(&supplier.accountInfo)
		newSuppliers = append(newSuppliers, supplier)
	}
	s.waitForNextBlock()
	s.preparedSuppliers = append(s.preparedSuppliers, newSuppliers...)
}
