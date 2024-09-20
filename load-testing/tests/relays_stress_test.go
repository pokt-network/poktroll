//go:build load

package tests

import (
	"context"
	"net/url"
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
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// The following constants are used to identify the different types of transactions,
// once committed, which are expected to be observed on-chain during the test.
// NB: The TxResult Events' #Type values are not prefixed with a slash,
// unlike TxResult Events' "action" attribute value.
const (
	EventActionMsgStakeApplication = "/poktroll.application.MsgStakeApplication"
	EventActionMsgStakeGateway     = "/poktroll.gateway.MsgStakeGateway"
	EventActionMsgStakeSupplier    = "/poktroll.supplier.MsgStakeSupplier"
	EventActionMsgCreateClaim      = "/poktroll.proof.MsgCreateClaim"
	EventActionMsgSubmitProof      = "/poktroll.proof.MsgSubmitProof"
	EventActionAppMsgUpdateParams  = "/poktroll.application.MsgUpdateParams"
	EventTypeRedelegation          = "poktroll.application.EventRedelegation"
)

// The following constants define the expected ordering of the actors when
// presented as rows in a table.
// NB: +1 to skip the header row.
const (
	applicationRowIdx = iota + 1
	gatewayRowIdx
	supplierRowIdx
)

// NB: +1 to skip the "actor" column.
const initialActorCountColIdx = iota + 1

// NB: +1 to skip the "actor" column.
const (
	actorIncrementAmountColIdx = iota + 1
	blocksPerIncrementColIdx
	maxAmountColIdx
)

// The current txClient implementation only supports online mode signing, which
// is simpler to implement since it is querying the signer account info from the
// blockchain node and abstracting the need to manually manage the sequence number.
// The sequence number is needed to ensure that the transactions are signed in the
// correct order and that the transactions are not replayed. See:
// * https://github.com/cosmos/cosmos-sdk/blob/main/proto/cosmos/tx/v1beta1/tx.proto#L164
// * https://github.com/cosmos/cosmos-sdk/blob/main/x/auth/client/tx.go#L59
// The load test sometimes fail to fetch the account information and retries are needed.
// By observing the number of retries needed in the test environment, signing always
// succeeded after the second retry a safe number of retries was chosen to be 3.
const signTxMaxRetries = 3

var (
	// maxConcurrentRequestLimit is the maximum number of concurrent requests that can be made.
	// By default, it is set to the number of logical CPUs available to the process.
	maxConcurrentRequestLimit = runtime.GOMAXPROCS(0)
	// fundingAccountAddress is the address of the account used to fund other accounts.
	fundingAccountAddress string
	// supplierStakeAmount is the amount of tokens to stake by suppliers.
	supplierStakeAmount sdk.Coin
	// gatewayStakeAmount is the amount of tokens to stake by gateways.
	gatewayStakeAmount sdk.Coin
	// testedServiceId is the service ID for that all applications and suppliers will
	// be using in this test.
	testedServiceId string
	// blockDuration is the duration of a block in seconds.
	// NB: This value SHOULD be equal to `timeout_propose` in `config.yml`.
	blockDuration = int64(2)
	// newTxEventSubscriptionQuery is the format string which yields a subscription
	// query to listen for on-chain Tx events.
	newTxEventSubscriptionQuery = "tm.event='Tx'"
	// eventsReplayClientBufferSize is the buffer size for the events replay client
	// for the subscriptions above.
	eventsReplayClientBufferSize = 100
	// relayPayloadFmt is the JSON-RPC request relayPayloadFmt to send a relay request.
	relayPayloadFmt = `{"jsonrpc":"2.0","method":"%s","params":[],"id":%d}`
	// relayRequestMethod is the method of the JSON-RPC request to be relayed.
	// Since the goal of the relay stress test is to stress request load, not network
	// bandwidth, a simple getHeight request is used.
	relayRequestMethod = "eth_blockNumber"
)

// relaysSuite is a test suite for the relays stress test.
// It tests the performance of the relays module by sending a number of relay requests
// concurrently to a network of applications, gateways, and suppliers.
// The test is parameterized by the number of applications, gateways, and suppliers to be staked,
// and the rate at which applications send relays.
type relaysSuite struct {
	gocuke.TestingT
	// ctx is the global context for the test suite.
	// It is canceled when the test suite is cleaned up causing all goroutines
	// and observables subscriptions to be canceled.
	ctx context.Context
	// cancelCtx is the cancel function for the global context.
	cancelCtx context.CancelFunc

	// blockClient notifies the test suite of new blocks committed.
	blockClient client.BlockClient
	// latestBlock is continuously updated with the latest committed block.
	latestBlock client.Block
	// sessionInfoObs is the observable that maps committed blocks to session information.
	// It is used to determine when to stake new actors and when they become active.
	sessionInfoObs observable.Observable[*sessionInfoNotif]
	// batchInfoObs is the observable mapping session information to batch information.
	// It is used to determine when to send a batch of relay requests to the network.
	batchInfoObs observable.Observable[*relayBatchInfoNotif]
	// newTxEventsObs is the observable that notifies the test suite of new
	// transactions committed on-chain.
	// It is used to check the results of the transactions sent by the test suite.
	newTxEventsObs observable.Observable[*types.TxResult]
	// txContext is the transaction context used to sign and send transactions.
	txContext client.TxContext
	// sharedParams is the shared on-chain parameters used in the test.
	// It is queried at the beginning of the test.
	sharedParams *sharedtypes.Params

	// numRelaysSent is the number of relay requests sent during the test.
	numRelaysSent atomic.Uint64
	// relayRatePerApp is the rate of relay requests sent per application per second.
	relayRatePerApp int64
	// relayCoinAmountCost is the amount of tokens (e.g. "upokt") a relay request costs.
	// It is equal to the tokenomics module's `compute_units_to_tokens_multiplier` parameter.
	relayCoinAmountCost int64

	// gatewayInitialCount is the number of active gateways at the start of the test.
	gatewayInitialCount int64
	// supplierInitialCount is the number of suppliers available at the start of the test.
	supplierInitialCount int64
	// appInitialCount is the number of active applications at the start of the test.
	appInitialCount int64

	// testStartHeight is the block height at which the test started.
	// It is used to calculate the progress of the test.
	testStartHeight int64
	testEndHeight   int64

	// relayLoadDurationBlocks is the duration in blocks it takes to send all relay requests.
	// After this duration, the test suite will stop sending relay requests, but will continue
	// to submit claims and proofs.
	// It is calculated as the longest duration of the three actor increments.
	relayLoadDurationBlocks int64

	// plans is the actor load test increment plans used to increment the actors during the test
	// and calculate the test duration.
	plans *actorLoadTestIncrementPlans

	// gatewayUrls is a map of gatewayAddress->URL representing the provisioned gateways.
	// These gateways are not staked yet but have their off-chain instance running
	// and ready to be staked and used in the test.
	// Since AppGateServers are pre-provisioned, and already assigned a signingAddress
	// and an URL to send relays to, the test suite does not create new ones but picks
	// from this list.
	// The max gateways used in the test must be less than or equal to the number of
	// provisioned gateways.
	gatewayUrls map[string]string
	// availableGatewayAddresses is the list of available gateway addresses to be used
	// in the test. It is populated from the gatewayUrls map.
	// It is used to ensure that the gateways are staked in the order they are provisioned.
	availableGatewayAddresses []string
	// suppliersUrls is a map of supplierOperatorAddress->URL representing the provisioned suppliers.
	// These suppliers are not staked yet but have their off-chain instance running
	// and ready to be staked and used in the test.
	// Since RelayMiners are pre-provisioned, and already assigned a signingAddress
	// and an URL, the test suite does not create new ones but picks from this list.
	// The max suppliers used in the test must be less than or equal to the number of
	// provisioned suppliers.
	suppliersUrls map[string]string
	// availableSupplierOperatorAddresses is the list of available supplier operator addresses to be used
	// in the test. It is populated from the suppliersUrls map.
	// It is used to ensure that the suppliers are staked in the order they are provisioned.
	// The same address is used as the owner and the operator address (i.e. custodial staking).
	availableSupplierOperatorAddresses []string

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
	// activeSuppliers is the list of suppliers that are currently staked and
	// ready to handle relay requests.
	activeSuppliers []*accountInfo

	// Number of claims and proofs observed on-chain during the test.
	currentProofCount int
	currentClaimCount int

	// expectedClaimsAndProofsCount is the expected number of claims and proofs
	// to be committed on-chain during the test.
	expectedClaimsAndProofsCount int

	// isEphemeralChain is a flag that indicates whether the test is expected to be
	// run on ephemeral chain setups like localnet or long living ones (i.e. Test/DevNet).
	isEphemeralChain bool
}

// accountInfo contains the account info needed to build and send transactions.
type accountInfo struct {
	// The address of the account available in the keyring used by the test.
	address       string
	amountToStake sdk.Coin
	// pendingMsgs is a list of messages that are pending to be sent by the account.
	// It is used to accumulate messages to be sent in a single transaction to avoid
	// sending multiple transactions across multiple blocks.
	pendingMsgs []sdk.Msg
}

func (ai *accountInfo) addPendingMsg(msg sdk.Msg) {
	ai.pendingMsgs = append(ai.pendingMsgs, msg)
}

// sessionInfoNotif is a struct containing the session information of a block.
type sessionInfoNotif struct {
	blockHeight             int64
	sessionNumber           int64
	sessionStartBlockHeight int64
	sessionEndBlockHeight   int64
}

// relayBatchInfoNotif is a struct containing the batch information used to calculate
// and schedule the relay requests to be sent.
type relayBatchInfoNotif struct {
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

func TestLoadRelaysSingleSupplier(t *testing.T) {
	gocuke.NewRunner(t, &relaysSuite{}).Path(filepath.Join(".", "relays_stress_single_suppier.feature")).Run()
}

func (s *relaysSuite) LocalnetIsRunning() {
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	// Cancel the context if this process is interrupted or exits.
	// Delete the keyring entries for the application accounts since they are
	// not persisted across test runs.
	signals.GoOnExitSignal(func() {
		for _, app := range append(s.activeApplications, s.preparedApplications...) {
			accAddress := sdk.MustAccAddressFromBech32(app.address)

			_ = s.txContext.GetKeyring().DeleteByAddress(accAddress)
		}
		s.cancelCtx()
	})

	s.Cleanup(func() {
		for _, app := range s.activeApplications {
			accAddress := sdk.MustAccAddressFromBech32(app.address)

			s.txContext.GetKeyring().DeleteByAddress(accAddress)
		}
		for _, app := range s.preparedApplications {
			accAddress, err := sdk.AccAddressFromBech32(app.address)
			require.NoError(s, err)

			s.txContext.GetKeyring().DeleteByAddress(accAddress)
		}
	})

	// Initialize the provisioned gateway and suppliers address->URL map that will
	// be populated from the load test manifest.
	s.gatewayUrls = make(map[string]string)
	s.suppliersUrls = make(map[string]string)

	// Parse the load test manifest.
	loadTestParams := s.initializeLoadTestParams()

	// Set the tested service ID from the load test manifest.
	testedServiceId = loadTestParams.ServiceId

	// If the test is run on a non-ephemeral chain, set the CometLocalTCPURL and
	// CometLocalWebsocketURL to the TestNetNode URL. These variables are used
	// by the testtx txClient to send transactions to the network.
	if !s.isEphemeralChain {
		testclient.CometLocalTCPURL = loadTestParams.TestNetNode

		webSocketURL, err := url.Parse(loadTestParams.TestNetNode)
		require.NoError(s, err)

		// TestNet nodes may be exposed over HTTPS, so adjust the scheme accordingly.
		if webSocketURL.Scheme == "https" {
			webSocketURL.Scheme = "wss"
		} else {
			webSocketURL.Scheme = "ws"
		}
		testclient.CometLocalWebsocketURL = webSocketURL.String() + "/websocket"

		// Update the block duration when running the test on a non-ephemeral chain.
		// TODO_TECHDEBT: Get the block duration value from the chain or the manifest.
		blockDuration = 60
	}

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
	s.relayCoinAmountCost = s.getRelayCost()

	// Setup the tx listener for on-chain events to check and assert on transactions results.
	s.setupTxEventListeners()

	// Initialize the funding account.
	s.initFundingAccount(loadTestParams.FundingAccountAddress)

	// Initialize the on-chain claims and proofs counter.
	s.countClaimAndProofs()

	// Query for the current shared on-chain params.
	s.querySharedParams(loadTestParams.TestNetNode)

	// Some suppliers may already be staked at genesis, ensure that staking during
	// this test succeeds by increasing the sake amount.
	minStakeAmount := s.getProvisionedActorsCurrentStakedAmount()
	supplierStakeAmount = sdk.NewCoin("upokt", math.NewInt(minStakeAmount+1))
	gatewayStakeAmount = sdk.NewCoin("upokt", math.NewInt(minStakeAmount+1))
}

func (s *relaysSuite) ARateOfRelayRequestsPerSecondIsSentPerApplication(appRPS string) {
	relayRatePerApp, err := strconv.ParseInt(appRPS, 10, 32)
	require.NoError(s, err)

	s.relayRatePerApp = relayRatePerApp
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	// Store the initial counts of the actors to be staked to be used later in the test,
	// when information about max actors to be staked is available.
	s.appInitialCount = table.Cell(applicationRowIdx, initialActorCountColIdx).Int64()
	// In the case of non-ephemeral chains, the gateway and supplier counts are
	// not under the test suite control and the initial counts are not stored.
	if !s.isEphemeralChain {
		return
	}

	s.gatewayInitialCount = table.Cell(gatewayRowIdx, initialActorCountColIdx).Int64()
	s.supplierInitialCount = table.Cell(supplierRowIdx, initialActorCountColIdx).Int64()
}

func (s *relaysSuite) MoreActorsAreStakedAsFollows(table gocuke.DataTable) {
	// Parse and validate the actor increment plans from the given step table.
	s.plans = s.parseActorLoadTestIncrementPlans(table)
	s.validateActorLoadTestIncrementPlans(s.plans)

	// The relay load duration is the longest duration of the three actor increments.
	// The duration of each actor is calculated as how many blocks it takes to
	// increment the actor count to the maximum.
	s.relayLoadDurationBlocks = s.plans.maxActorBlocksToFinalIncrementEnd()

	if s.isEphemeralChain {
		// Adjust the max delegations parameter to the max gateways to permit all
		// applications to delegate to all gateways.
		// This is to ensure that requests are distributed evenly across all gateways
		// at any given time.
		s.sendAdjustMaxDelegationsParamTx(s.plans.gateways.maxActorCount)
		s.waitForTxsToBeCommitted()
		s.ensureUpdatedMaxDelegations(s.plans.gateways.maxActorCount)
	}

	// Fund all the provisioned suppliers and gateways since their addresses are
	// known and they are not created on the fly, while funding only the initially
	// created applications.
	fundedSuppliers, fundedGateways, fundedApplications := s.sendFundAvailableActorsTx()
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
	s.ensureStakedActors(txResults, EventActionMsgStakeSupplier, suppliers)
	s.ensureStakedActors(txResults, EventActionMsgStakeGateway, gateways)
	s.ensureStakedActors(txResults, EventActionMsgStakeApplication, applications)

	logger.Info().Msg("Actors staked")

	// Update the list of staked suppliers.
	s.activeSuppliers = append(s.activeSuppliers, suppliers...)

	// In the case of non-ephemeral chain load tests, the available gateways are
	// not incrementally staked, but are already staked and delegated to, add all
	// of them to the list of active gateways at the beginning of the test.
	if !s.isEphemeralChain {
		gateways = s.populateWithKnownGateways()
	}

	// Delegate the initial applications to the initial gateways
	s.sendDelegateInitialAppsTxs(applications, gateways)
	txResults = s.waitForTxsToBeCommitted()
	s.ensureDelegatedApps(txResults, applications, gateways)

	logger.Info().Msg("Apps delegated")

	// Applications and gateways are now ready and will be active in the next session.
	s.preparedApplications = append(s.preparedApplications, applications...)
	s.preparedGateways = append(s.preparedGateways, gateways...)

	// relayBatchInfoObs maps session information to batch information used to schedule
	// the relay requests to be sent on the current block.
	relayBatchInfoObs, relayBatchInfoPublishCh := channel.NewReplayObservable[*relayBatchInfoNotif](s.ctx, 5)
	s.batchInfoObs = relayBatchInfoObs

	// sessionInfoObs asynchronously maps committed blocks to a notification which
	// includes the session number and the start and end block heights of the session.
	// It runs at the same frequency as committed blocks (i.e. 1:1).
	s.sessionInfoObs = channel.Map(
		s.ctx,
		s.blockClient.CommittedBlocksSequence(s.ctx),
		s.mapSessionInfoForLoadTestDurationFn(relayBatchInfoPublishCh),
	)

	// stakingSuppliersAndGatewaysObs notifies when actors are to be incremented, after staking suppliers & gateways.
	stakingSuppliersAndGatewaysObs := channel.Map(
		s.ctx,
		s.sessionInfoObs,
		s.mapSessionInfoWhenStakingNewSuppliersAndGatewaysFn(),
	)

	// stakedAndDelegatingObs notifies when staking and delegation transactions are sent.
	stakedAndDelegatingObs := channel.Map(
		s.ctx,
		stakingSuppliersAndGatewaysObs,
		s.mapStakingInfoWhenStakingAndDelegatingNewApps,
	)

	// When staking and delegation transactions are sent, wait for them to be committed
	// before adding the new actors to the list of prepared actors to be activated in
	// the next session.
	channel.ForEach(
		s.ctx,
		stakedAndDelegatingObs,
		s.forEachStakedAndDelegatedAppPrepareApp,
	)
}

// ALoadOfConcurrentRelayRequestsAreSentFromTheApplications sends batches of relay
// requests for each active application to one active gateway (round-robin; per relay).
// Relays within a batch are distributed over time to match the configured rate
// (relays per second).
func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentFromTheApplications() {
	// Asynchronously send relay request batches for each batch info notification.
	channel.ForEach(s.ctx, s.batchInfoObs, s.forEachRelayBatchSendBatch)

	// Block the feature step until the test is done.
	<-s.ctx.Done()
}
func (s *relaysSuite) TheCorrectPairsCountOfClaimAndProofMessagesShouldBeCommittedOnchain() {
	logger.Info().
		Int("claims", s.currentClaimCount).
		Int("proofs", s.currentProofCount).
		Msg("Claims and proofs count")

	require.Equal(s,
		s.currentClaimCount,
		s.currentProofCount,
		"claims and proofs count mismatch",
	)

	// TODO_TECHDEBT: The current counting mechanism for the expected claims and proofs
	// is not accurate. The expected claims and proofs count should be calculated based
	// on a function of(time_per_block, num_blocks_per_session) -> num_claims_and_proofs.
	// The reason (time_per_block) is one of the parameters is because claims and proofs
	// are removed from the on-chain state after sessions are settled, only leaving
	// events behind. The final solution needs to either account for this timing
	// carefully (based on sessions that have passed), or be event driven using
	// a replay client of on-chain messages and/or events.
	//require.Equal(s,
	//	s.expectedClaimsAndProofsCount,
	//	s.currentProofCount,
	//	"unexpected claims and proofs count",
	//)
}
