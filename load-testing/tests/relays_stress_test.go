package tests

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client"
	blocktypes "github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/sync2"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	MsgStakeApplication = "/poktroll.application.MsgStakeApplication"
	MsgStakeGateway     = "/poktroll.gateway.MsgStakeGateway"
	MsgStakeSupplier    = "/poktroll.supplier.MsgStakeSupplier"
	AppMsgUpdateParams  = "/poktroll.application.MsgUpdateParams"
	EventRedelegation   = "poktroll.application.EventRedelegation"
)

var (
	// maxConcurrentRequestLimit is the maximum number of concurrent requests that can be made.
	// By default, it is set to the number of logical CPUs available to the process.
	maxConcurrentRequestLimit = runtime.GOMAXPROCS(0)
	// fundingAccountKeyName is the key name of the account used to fund other accounts.
	fundingAccountKeyName = "pnf"
	// stakeAmount is the amount of tokens to stake by suppliers and gateways.
	stakeAmount sdk.Coin
	// usedService is the service ID for that all applications and suppliers will
	// be using in this test.
	usedService = &sharedtypes.Service{Id: "anvil"}
	// loadTestManifestPath is the path to the load test manifest file.
	// It is used to initialize the provisioned gateways and suppliers used in the test.
	// TODO_TECHDEBT: Get the path of the load test manifest from CLI flags.
	loadTestManifestPath = "../../loadtest_manifest.yaml"
	// blockDuration is the duration of a block in seconds.
	blockDuration = int64(2)
	// newTxEventSubscriptionQuery is the format string which yields a subscription
	// query to listen for on-chain Tx events.
	newTxEventSubscriptionQuery = "tm.event='Tx'"
	// eventsReplayClientBufferSize is the buffer size for the events replay client
	// for the subscriptions above.
	eventsReplayClientBufferSize = 100
	// relayPayload is the JSON-RPC request relayPayload to send a relay request.
	relayPayload = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
)

// relaysSuite is a test suite for the relays stress test.
// It tests the performance of the relays module by sending a number of relay requests
// concurrently to a network of applications, gateways, and suppliers.
// The test is parameterized by the number of applications, gateways, and suppliers to be staked,
// and the rate at which applications send relays.
type relaysSuite struct {
	gocuke.TestingT
	// ctx is the global context for the test suite.
	// It is cancelled when the test suite is cleaned up causing all goroutines
	// and observables subscriptions to be cancelled.
	ctx context.Context
	// cancelCtx is the cancel function for the global context.
	cancelCtx context.CancelFunc

	// blockClient notifies the test suite of new blocks committed.
	blockClient client.BlockClient
	latestBlock *blocktypes.CometNewBlockEvent
	// sessionInfoObs is the observable that maps committed blocks to session information.
	// It is used to determine when to stake new actors and when they become active.
	sessionInfoObs observable.Observable[*sessionInfoNotif]
	// batchInfoObs is the observable mapping session information to batch information.
	// It is used to determine when to send a batch of relay requests to the network.
	batchInfoObs observable.Observable[*batchInfoNotif]
	// newBlocksEventsClient is the observable that notifies the test suite of new
	// about new transactions committed on-chain.
	// It is used to check the results of the transactions sent by the test suite.
	newTxEventsObs observable.Observable[*types.TxResult]
	// txContext is the transaction context used to sign and send transactions.
	txContext client.TxContext

	// relaysSent is the number of relay requests sent during the test.
	relaysSent atomic.Uint64
	// relayRatePerApp is the rate of relay requests sent per application per second.
	relayRatePerApp int64
	// relayCost is the cost of a relay request.
	relayCost int64

	// gatewayInitialCount is the number of active gateways at the start of the test.
	gatewayInitialCount int64
	// supplierInitialCount is the number of suppliers available at the start of the test.
	supplierInitialCount int64
	// appInitialCount is the number of active applications at the start of the test.
	appInitialCount int64

	// startBlockHeight is the block height at which the test started.
	// It is used to calculate the progress of the test.
	startBlockHeight int64
	// testDurationBlocks is the duration of the test in blocks.
	// It is used to determine when the test is done.
	// It is calculated as the longest duration of the three actor increments.
	testDurationBlocks int64

	// gatewayUrls is a map of gatewayKeyName->URL representing the provisioned gateways.
	// These gateways are not staked yet but have their off-chain instance running
	// and ready to be staked and used in the test.
	// Since AppGateServers are pre-provisioned, and already assigned a signingKeyName
	// and an URL to send relays to, the test suite does not create new ones but picks
	// from this list.
	// The max gateways used in the test must be less than or equal to the number of
	// provisioned gateways.
	gatewayUrls map[string]*url.URL
	// suppliersUrls is a map of supplierKeyName->URL representing the provisioned suppliers.
	// These suppliers are not staked yet but have their off-chain instance running
	// and ready to be staked and used in the test.
	// Since RelayMiners are pre-provisioned, and already assigned a signingKeyName
	// and an URL, the test suite does not create new ones but picks from this list.
	// The max suppliers used in the test must be less than or equal to the number of
	// provisioned suppliers.
	suppliersUrls map[string]*url.URL

	// fundingAccountInfo is the account entry corresponding to the fundingAccountKeyName.
	// It is used to send transactions to fund other accounts.
	fundingAccountInfo *accountInfo
	// preparedGateways is the list of gateways that are already staked, delegated
	// to and ready to be used in the next session.
	preparedGateways []*accountInfo
	// preparedApplications is the list of applications that are already staked,
	// delegated and ready to be used in the next session.
	preparedApplications []*accountInfo
	// activeGateways is the list of gateways that are currently staked, delegated
	// to and used by the applications to send relay requests to the network.
	activeGateways []*accountInfo
	// activeApplications is the list of applications that are currently staked,
	// delegated and sending relays to the gateways.
	activeApplications []*accountInfo
	// stakedSuppliers is the list of suppliers that are currently staked and
	// ready to handle relay requests.
	stakedSuppliers []*accountInfo
}

// accountInfo contains the account info needed to build and send transactions.
type accountInfo struct {
	// keyName is the key name of the account available in the keyring used by the test.
	keyName       string
	accAddress    sdk.AccAddress
	amountToStake sdk.Coin
	// pendingMsgs is a list of messages that are pending to be sent by the account.
	// It is used to accumulate messages to be sent in a single transaction to avoid
	// sending multiple transactions across multiple blocks.
	pendingMsgs []sdk.Msg
}

// sessionInfoNotif is a struct containing the session information of a block.
type sessionInfoNotif struct {
	blockHeight             int64
	sessionNumber           int64
	sessionStartBlockHeight int64
	sessionEndBlockHeight   int64
}

// batchInfoNotif is a struct containing the batch information used to calculate
// and schedule the relay requests to be sent.
type batchInfoNotif struct {
	sessionInfoNotif
	prevBatchTime time.Time
	nextBatchTime time.Time
	appAccounts   []*accountInfo
	gateways      []*accountInfo
}

type stakingInfo struct {
	sessionInfoNotif
	newApps      []*accountInfo
	newGateways  []*accountInfo
	newSuppliers []*accountInfo
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

	// Initialize the provisioned gateway and suppliers keyName->URL map that will
	// be populated from the load test manifest.
	s.gatewayUrls = make(map[string]*url.URL)
	s.suppliersUrls = make(map[string]*url.URL)

	// Set up the blockClient that will be notifying the suite about the committed blocks.
	s.blockClient = testblock.NewLocalnetClient(s.ctx, s.TestingT.(*testing.T))
	channel.ForEach(
		s.ctx,
		s.blockClient.CommittedBlocksSequence(s.ctx),
		func(ctx context.Context, block client.Block) {
			s.latestBlock = block.(*blocktypes.CometNewBlockEvent)
		},
	)
	<-s.blockClient.CommittedBlocksSequence(s.ctx).Subscribe(s.ctx).Ch()

	// Setup the txClient that will be used to send transactions to the network.
	s.txContext = testtx.NewLocalnetContext(s.TestingT.(*testing.T))

	// Get the relay cost from the tokenomics module.
	s.relayCost = s.getRelayCost()

	// Setup the tx listener for on-chain events to check and assert on transactions results.
	s.setupTxEventListeners()

	// Initialize the funding account.
	s.initFundingAccount(fundingAccountKeyName)

	// Initialize the provisioned gateways and suppliers from the load test manifest.
	s.initializeProvisionedActors()

	// Some suppliers may already be staked at genesis, ensure that staking during
	// this test succeeds by increasing the sake amount.
	supplierStakeAmount := s.getSuppliersCurrentStakedAmount()
	stakeAmount = sdk.NewCoin("upokt", math.NewInt(supplierStakeAmount+1))
}

func (s *relaysSuite) ARateOfRelayRequestsPerSecondIsSentPerApplication(appRPS string) {
	relayRatePerApp, err := strconv.ParseInt(appRPS, 10, 32)
	require.NoError(s, err)

	s.relayRatePerApp = relayRatePerApp
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	// Store the initial counts of the actors to be staked to be used later in the test,
	// when information about max actors to be staked is available.
	s.supplierInitialCount = table.Cell(3, 1).Int64()
	s.gatewayInitialCount = table.Cell(1, 1).Int64()
	s.appInitialCount = table.Cell(2, 1).Int64()
}

func (s *relaysSuite) MoreActorsAreStakedAsFollows(table gocuke.DataTable) {
	gatewayInc := table.Cell(1, 1).Int64()
	gatewayBlockIncRate := table.Cell(1, 2).Int64()
	require.Truef(s,
		gatewayBlockIncRate%keeper.NumBlocksPerSession == 0,
		"gateway increment rate must be a multiple of the session length",
	)
	maxGateways := table.Cell(1, 3).Int64()
	require.Truef(s,
		len(s.gatewayUrls) >= int(maxGateways),
		"provisioned gateways must be greater or equal than the max gateways to be staked",
	)

	supplierInc := table.Cell(3, 1).Int64()
	supplierBlockIncRate := table.Cell(3, 2).Int64()
	require.Truef(s,
		supplierBlockIncRate%keeper.NumBlocksPerSession == 0,
		"supplier increment rate must be a multiple of the session length",
	)
	maxSuppliers := table.Cell(3, 3).Int64()
	require.Truef(s,
		len(s.suppliersUrls) >= int(maxSuppliers),
		"provisioned suppliers must be greater or equal than the max suppliers to be staked",
	)

	appInc := table.Cell(2, 1).Int64()
	appBlockIncRate := table.Cell(2, 2).Int64()
	maxApps := table.Cell(2, 3).Int64()
	require.Truef(s,
		appBlockIncRate%keeper.NumBlocksPerSession == 0,
		"app increment rate must be a multiple of the session length",
	)

	// The test duration is the longest duration of the three actor increments.
	// The duration of each actor is calculated as how many blocks it takes to
	// increment the actor count to the maximum.
	s.testDurationBlocks = math.Max(
		maxGateways/gatewayInc*gatewayBlockIncRate,
		maxApps/appInc*appBlockIncRate,
		maxSuppliers/supplierInc*supplierBlockIncRate,
	)

	// Adjust the max delegations parameter to the max gateways to permit all
	// applications to delegate to all gateways.
	// This is to ensure that requests are distributed evenly across all gateways
	// at any given time.
	s.sendAdjustMaxDelegationsParamTx(maxGateways)
	s.waitForTxsToBeCommitted()
	s.ensureUpdatedMaxDelegations(maxGateways)

	// Fund all the provisioned suppliers and gateways since their addresses are
	// known and they are not created on the fly, while funding only the initially
	// created applications.
	fundedSuppliers, fundedGateways, fundedApplications := s.sendFundAvailableActorsTx(maxSuppliers, maxGateways)
	// Funding messages are sent in a single transaction by the funding account,
	// only one transaction is expected to be committed.
	txResults := s.waitForTxsToBeCommitted()
	s.ensureFundedActors(txResults, fundedSuppliers)
	s.ensureFundedActors(txResults, fundedGateways)
	s.ensureFundedActors(txResults, fundedApplications)

	logger.Info().Msg("Actors funded")

	// The initial actors are the first actors to stake.
	suppliers := fundedSuppliers[:s.supplierInitialCount]
	gateways := fundedGateways[:s.gatewayInitialCount]
	applications := fundedApplications[:s.appInitialCount]

	s.sendInitialActorsStakeMsgs(suppliers, gateways, applications)
	txResults = s.waitForTxsToBeCommitted()
	s.ensureStakedActors(txResults, MsgStakeSupplier, suppliers)
	s.ensureStakedActors(txResults, MsgStakeGateway, gateways)
	s.ensureStakedActors(txResults, MsgStakeApplication, applications)

	logger.Info().Msg("Actors staked")

	// Update the list of staked suppliers.
	s.stakedSuppliers = append(s.stakedSuppliers, suppliers...)

	// Delegate the initial applications to the initial gateways
	s.sendDelegateInitialAppsTxs(applications, gateways)
	txResults = s.waitForTxsToBeCommitted()
	s.ensureDelegatedApps(txResults, applications, gateways)

	logger.Info().Msg("Apps delegated")

	// Applications and gateways are now ready and will be active in the next session.
	s.preparedApplications = append(s.preparedApplications, applications...)
	s.preparedGateways = append(s.preparedGateways, gateways...)

	// The test suite is initially waiting for the next session to start.
	waitingForFirstSession := true
	var prevBatchTime time.Time

	// batchInfoObs maps session information to batch information used to schedule
	// the relay requests to be sent on the current block.
	batchInfoObs, batchInfoPublishCh := channel.NewReplayObservable[*batchInfoNotif](s.ctx, 5)
	s.batchInfoObs = batchInfoObs

	// sessionInfoObs maps committed blocks to a notification which includes the
	// session number and the start and end block heights of the session.
	// It runs at the same frequency as committed blocks (i.e. 1:1).
	s.sessionInfoObs = channel.Map(s.ctx, s.blockClient.CommittedBlocksSequence(s.ctx),
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

			infoLogger := logger.Info().
				Int64("session_num", sessionInfo.sessionNumber).
				Int64("block_height", block.Height())

			// If the test has not started and the current block is not the first block
			// of the session, wait for the next session to start.
			if waitingForFirstSession && blockHeight != sessionInfo.sessionStartBlockHeight {
				countDownToTestStart := sessionInfo.sessionEndBlockHeight - blockHeight + 1
				infoLogger.Msgf(
					"waiting for next session to start: in %d blocks",
					countDownToTestStart,
				)

				// The test is not to be started yet, skip the notification to the downstream
				// observables until the first block of the next session is reached.
				return nil, true
			}

			// If the test has not started, set the start block height to the current block height.
			// As soon as the test start, s.startBlockHeight will no longer be updated.
			// It is updated only once at the start of the test.
			if waitingForFirstSession {
				s.startBlockHeight = blockHeight
			}

			// Mark the test as started.
			waitingForFirstSession = false

			// Log the test progress.
			infoLogger.Msgf(
				"test progress blocks: %d/%d",
				blockHeight-s.startBlockHeight, s.testDurationBlocks,
			)

			// If the test duration is reached, cancel the context to stop the test.
			if blockHeight >= s.startBlockHeight+s.testDurationBlocks {
				logger.Info().Msg("Test done, cancelling scenario context")
				s.cancelCtx()

				return nil, true
			}

			// If the current block is the start of any new session, activate the prepared
			// actors to be used in the current session.
			s.activatePreparedActors(sessionInfo)

			now := time.Now()

			// Inform the relay sending observable of the active applications that
			// will be sending relays and the gateways that will be receiving them.
			batchInfoPublishCh <- &batchInfoNotif{
				sessionInfoNotif: *sessionInfo,
				prevBatchTime:    prevBatchTime,
				nextBatchTime:    now,
				appAccounts:      s.activeApplications,
				gateways:         s.activeGateways,
			}

			// Update prevBatchTime after this iteration completes.
			prevBatchTime = now

			// Forward the session info notification to the downstream observables.
			return sessionInfo, false
		},
	)

	// When the test starts, each block is processed to determine if any new actors
	// need to be staked or activated.
	stakingObs := channel.Map(s.ctx, s.sessionInfoObs,
		func(ctx context.Context, notif *sessionInfoNotif) (*stakingInfo, bool) {
			// Check if any new actors need to be staked **for use in the next session**.
			var newSuppliers []*accountInfo
			stakedSuppliers := int64(len(s.stakedSuppliers))
			if s.shouldIncrementSupplier(notif, supplierBlockIncRate, stakedSuppliers, maxSuppliers) {
				newSuppliers = s.sendStakeSuppliersTxs(notif, supplierInc, maxSuppliers)
			}

			var newGateways []*accountInfo
			activeGateways := int64(len(s.activeGateways))
			if s.shouldIncrementActor(notif, gatewayBlockIncRate, activeGateways, maxGateways) {
				newGateways = s.sendStakeGatewaysTxs(notif, gatewayInc, maxGateways)
			}

			var newApps []*accountInfo
			activeApps := int64(len(s.activeApplications))
			if s.shouldIncrementActor(notif, appBlockIncRate, activeApps, maxApps) {
				newApps = s.sendFundNewAppsTx(notif, appInc, maxApps)
			}

			// If no need to be processed in this block, skip the rest of the process.
			if len(newApps) == 0 && len(newGateways) == 0 && len(newSuppliers) == 0 {
				return nil, true
			}

			return &stakingInfo{
				sessionInfoNotif: *notif,
				newApps:          newApps,
				newGateways:      newGateways,
				newSuppliers:     newSuppliers,
			}, false
		},
	)

	stakedAndDelegatingObs := channel.Map(s.ctx, stakingObs,
		func(ctx context.Context, notif *stakingInfo) (*stakingInfo, bool) {
			// Ensure that new gateways and suppliers are staked.
			// Ensure that new applications are funded and have an account entry on-chain
			// so that they can stake and delegate in the next block.
			// The number of transactions to be committed is the sum of the number of new
			// gateways, suppliers and a single transaction to fund all new applications.
			txResults = s.waitForTxsToBeCommitted()
			s.ensureFundedActors(txResults, notif.newApps)
			s.ensureStakedActors(txResults, MsgStakeGateway, notif.newGateways)
			s.ensureStakedActors(txResults, MsgStakeSupplier, notif.newSuppliers)

			// Update the list of staked suppliers.
			s.stakedSuppliers = append(s.stakedSuppliers, notif.newSuppliers...)

			// If no apps or gateways are to be staked, skip the rest of the process.
			if len(notif.newApps) == 0 && len(notif.newGateways) == 0 {
				return nil, true
			}

			s.sendStakeAndDelegateAppsTxs(&notif.sessionInfoNotif, notif.newApps, notif.newGateways)

			return notif, false
		},
	)

	channel.ForEach(s.ctx, stakedAndDelegatingObs,
		func(ctx context.Context, notif *stakingInfo) {
			// Wait for the next block to commit staking and delegation transactions
			// and be able to send relay requests evenly distributed across all gateways.
			txResults = s.waitForTxsToBeCommitted()
			s.ensureStakedActors(txResults, MsgStakeApplication, notif.newApps)
			s.ensureDelegatedApps(txResults, s.activeApplications, notif.newGateways)
			s.ensureDelegatedApps(txResults, notif.newApps, notif.newGateways)
			s.ensureDelegatedApps(txResults, notif.newApps, s.activeGateways)

			// Add the new actors to the list of prepared actors to be activated in
			// the next session.
			s.preparedApplications = append(s.preparedApplications, notif.newApps...)
			s.preparedGateways = append(s.preparedGateways, notif.newGateways...)
		},
	)
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentFromTheApplications() {
	// Limit the number of concurrent requests to maxConcurrentRequestLimit.
	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)

	channel.ForEach(s.ctx, s.batchInfoObs,
		func(ctx context.Context, batchInfo *batchInfoNotif) {
			// Calculate the relays per second as the number of active applications
			// each sending relayRatePerApp relays per second.
			relaysPerSec := len(batchInfo.appAccounts) * int(s.relayRatePerApp)
			// Determine the interval between each relay request.
			relayInterval := time.Second / time.Duration(relaysPerSec)

			batchWaitGroup := new(sync.WaitGroup)
			batchWaitGroup.Add(relaysPerSec * int(blockDuration))

			for i := 0; i < relaysPerSec*int(blockDuration); i++ {
				batchLimiter.Go(s.ctx, func() {

					relaysSent := s.relaysSent.Add(1) - 1

					// Send the relay request.
					s.sendRelay(relaysSent)

					//logger.Debug().
					//	Int64("session_num", batchInfo.sessionNumber).
					//	Int64("block_height", batchInfo.blockHeight).
					//	Str("app", appKeyName).
					//	Str("gw", gwKeyName).
					//	Int("total_apps", len(batchInfo.appAccounts)).
					//	Int("total_gws", len(batchInfo.gateways)).
					//	Str("time", time.Now().Format(time.RFC3339Nano)).
					//	Msgf("sending relay #%d", relaysSent)

					batchWaitGroup.Done()
				})

				// Sleep for the interval between each relay request.
				time.Sleep(relayInterval)
			}

			// Wait until all relay requests in the batch are sent.
			batchWaitGroup.Wait()
		},
	)

	// Block the feature step until the test is done.
	<-s.ctx.Done()
}
