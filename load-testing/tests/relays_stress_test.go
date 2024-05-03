package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
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
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/sync2"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	MsgStakeApplication = "/poktroll.application.MsgStakeApplication"
	MsgStakeGateway     = "/poktroll.gateway.MsgStakeGateway"
	MsgStakeSupplier    = "/poktroll.supplier.MsgStakeSupplier"
	MsgCreateClaim      = "/poktroll.proof.MsgCreateClaim"
	MsgSubmitProof      = "/poktroll.proof.MsgSubmitProof"
	AppMsgUpdateParams  = "/poktroll.application.MsgUpdateParams"
	EventRedelegation   = "poktroll.application.EventRedelegation"

	signTxMaxRetries = 3
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
	latestBlock client.Block
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
	gatewayUrls map[string]string
	// suppliersUrls is a map of supplierKeyName->URL representing the provisioned suppliers.
	// These suppliers are not staked yet but have their off-chain instance running
	// and ready to be staked and used in the test.
	// Since RelayMiners are pre-provisioned, and already assigned a signingKeyName
	// and an URL, the test suite does not create new ones but picks from this list.
	// The max suppliers used in the test must be less than or equal to the number of
	// provisioned suppliers.
	suppliersUrls map[string]string

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

	// Number of claims and proofs observed on-chain during the test.
	currentProofCount int
	currentClaimCount int

	// expectedClaimsAndProofsCount is the expected number of claims and proofs
	// to be committed on-chain during the test.
	expectedClaimsAndProofsCount int
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

type stakingInfoNotif struct {
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
	s.gatewayUrls = make(map[string]string)
	s.suppliersUrls = make(map[string]string)

	// Set up the blockClient that will be notifying the suite about the committed blocks.
	s.blockClient = testblock.NewLocalnetClient(s.ctx, s.TestingT.(*testing.T))
	channel.ForEach(
		s.ctx,
		s.blockClient.CommittedBlocksSequence(s.ctx),
		func(ctx context.Context, block client.Block) {
			s.latestBlock = block
		},
	)

	// Setup the txContext that will be used to send transactions to the network.
	s.txContext = testtx.NewLocalnetContext(s.TestingT.(*testing.T))

	// Get the relay cost from the tokenomics module.
	s.relayCost = s.getRelayCost()

	// Setup the tx listener for on-chain events to check and assert on transactions results.
	s.setupTxEventListeners()

	// Initialize the funding account.
	s.initFundingAccount(fundingAccountKeyName)

	// Initialize the provisioned gateways and suppliers from the load test manifest.
	s.initializeProvisionedActors()

	// Initialize the on-chain claims and proofs counter.
	s.countClaimAndProofs()

	// Some suppliers may already be staked at genesis, ensure that staking during
	// this test succeeds by increasing the sake amount.
	minStakeAmount := s.getProvisionedActorsCurrentStakedAmount()
	stakeAmount = sdk.NewCoin("upokt", math.NewInt(minStakeAmount+1))
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
	plan := actorPlans{
		gateways: actorPlan{
			initialAmount:   s.gatewayInitialCount,
			incrementAmount: table.Cell(1, 1).Int64(),
			incrementRate:   table.Cell(1, 2).Int64(),
			maxAmount:       table.Cell(1, 3).Int64(),
		},
		apps: actorPlan{
			initialAmount:   s.appInitialCount,
			incrementAmount: table.Cell(2, 1).Int64(),
			incrementRate:   table.Cell(2, 2).Int64(),
			maxAmount:       table.Cell(2, 3).Int64(),
		},
		suppliers: actorPlan{
			initialAmount:   s.supplierInitialCount,
			incrementAmount: table.Cell(3, 1).Int64(),
			incrementRate:   table.Cell(3, 2).Int64(),
			maxAmount:       table.Cell(3, 3).Int64(),
		},
	}

	s.validateActorPlans(&plan)

	// The test duration is the longest duration of the three actor increments.
	// The duration of each actor is calculated as how many blocks it takes to
	// increment the actor count to the maximum.
	s.testDurationBlocks = plan.maxDurationBlocks()

	// Adjust the max delegations parameter to the max gateways to permit all
	// applications to delegate to all gateways.
	// This is to ensure that requests are distributed evenly across all gateways
	// at any given time.
	s.sendAdjustMaxDelegationsParamTx(plan.gateways.maxAmount)
	s.waitForTxsToBeCommitted()
	s.ensureUpdatedMaxDelegations(plan.gateways.maxAmount)

	// Fund all the provisioned suppliers and gateways since their addresses are
	// known and they are not created on the fly, while funding only the initially
	// created applications.
	fundedSuppliers, fundedGateways, fundedApplications := s.sendFundAvailableActorsTx(&plan)
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

	// batchInfoObs maps session information to batch information used to schedule
	// the relay requests to be sent on the current block.
	batchInfoObs, batchInfoPublishCh := channel.NewReplayObservable[*batchInfoNotif](s.ctx, 5)
	s.batchInfoObs = batchInfoObs

	// sessionInfoObs asynchronously maps committed blocks to a notification which
	// includes the session number and the start and end block heights of the session.
	// It runs at the same frequency as committed blocks (i.e. 1:1).
	s.sessionInfoObs = channel.Map(
		s.ctx,
		s.blockClient.CommittedBlocksSequence(s.ctx),
		s.mapSessionInfoFn(batchInfoPublishCh),
	)

	// stakingObs asynchronously maps session information to a set of newly staked
	// actor accounts, only notifying when new actors were staked and skipping otherwise.
	// It stakes new suppliers & gateways but only funds new applications as they can't be
	// delegated until after the respective gateway stake txs have been committed.
	// It receives at the same frequency as committed blocks (i.e. 1:1) but only sends
	// conditionally as described here.
	stakingObs := channel.Map(s.ctx, s.sessionInfoObs, s.mapStakingInfoFn(plan))

	// stakedAndDelegatingObs asynchronously maps over the staking info, notified
	// when one or more actors have been newly staked. For each notification received,
	// it waits for the new actors' staking/funding txs to be committed before sending
	// staking & delegation txs for new applications.
	stakedAndDelegatingObs := channel.Map(s.ctx, stakingObs,
		func(ctx context.Context, notif *stakingInfoNotif) (*stakingInfoNotif, bool) {
			// Ensure that new gateways and suppliers are staked.
			// Ensure that new applications are funded and have an account entry on-chain
			// so that they can stake and delegate in the next block.
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

	// When staking and delegation transactions are sent, wait for them to be committed
	// before adding the new actors to the list of prepared actors to be activated in
	// the next session.
	channel.ForEach(s.ctx, stakedAndDelegatingObs,
		func(ctx context.Context, notif *stakingInfoNotif) {
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

	channel.ForEach(s.ctx, s.batchInfoObs, s.sendRelayBatchFn(batchLimiter))

	// Block the feature step until the test is done.
	<-s.ctx.Done()
}

func (s *relaysSuite) TheCorrectPairsCountOfClaimAndProofMessagesShouldBeCommittedOnchain() {
	require.Equal(s, s.currentClaimCount, s.currentProofCount, "claims and proofs count mismatch")
	require.Equal(s, s.expectedClaimsAndProofsCount, s.currentProofCount, "unexpected claims and proofs count")
}

func (ai *accountInfo) addPendingMsg(msg sdk.Msg) {
	ai.pendingMsgs = append(ai.pendingMsgs, msg)
}
