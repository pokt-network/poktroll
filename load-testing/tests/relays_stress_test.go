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
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
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
	// TODO_TECHDEBT: Make sure that applications have enough funds to pay for relays
	// given the amount of relays they are expected to send.
	fundingAmount = sdk.NewCoin("upokt", math.NewInt(10000000))
	// stakeAmount is the amount of tokens to stake for suppliers and gateways.
	stakeAmount = sdk.NewCoin("upokt", math.NewInt(2000))
	// applicationStakeAmount is the amount of tokens to stake for an application.
	// This amount should be enough to cover the cost of relays.
	applicationStakeAmount = sdk.NewCoin("upokt", math.NewInt(100000))
	// anvilService is the service ID for the Anvil service that all applications
	// and suppliers will be using in this test.
	anvilService = &sharedtypes.Service{Id: "anvil"}
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

	// waitingForFirstSession is a flag indicating whether the test is waiting for
	// the first session to start.
	waitingForFirstSession bool
	// startBlockHeight is the block height at which the test started.
	startBlockHeight int64

	// provisionedGateways is the list of provisioned gateways.
	// These are the gateways that are available to be staked.
	// Since AppGateServers are pre-provisioned, and already assigned a signingKeyName
	// and exposedServerAddress, the test suite does not create new ones but pick
	// from this list.
	provisionedGateways []*provisionedOffChainActor
	// provisionedSuppliers is the list of provisioned suppliers.
	// These are the suppliers that are available to be staked.
	// Since RelayMiners are pre-provisioned, and already assigned a signingKeyName
	// and exposedServerAddress, the test suite does not create new ones but pick
	// from this list and use its information to create StakeSupplierMsgs
	provisionedSuppliers []*provisionedOffChainActor

	// preparedGateways is the list of gateways that are already staked and ready
	// to be used in the next session.
	// They are segregated from activeGateways to avoid sending relay requests
	// to them since the delegation will be active in the next session.
	preparedGateways []*provisionedOffChainActor
	// preparedApplications is the list of applications that are already staked and ready
	// to be used in the next session.
	// They are segregated from activeApplications to avoid sending relay requests
	// from them since their delegations will be active in the next session.
	preparedApplications []*accountInfo
	// preparedSuppliers is the list of suppliers that are already staked and ready
	// to be used in the next session.
	preparedSuppliers []*provisionedOffChainActor

	// activeGateways is the list of gateways that are currently staked and active.
	// They are used to send relay requests to the staked suppliers.
	activeGateways []*provisionedOffChainActor
	// activeApplications is the list of applications that are currently staked and
	// used to send relays to the gateways they delegated to.
	activeApplications []*accountInfo
	// activeSuppliers is the list of suppliers that are currently staked and
	// ready to handle relay requests.
	activeSuppliers []*provisionedOffChainActor
}

// accountInfo is a struct containing the information of an account.
type accountInfo struct {
	keyName     string
	accAddress  sdk.AccAddress
	privKey     *secp256k1.PrivKey
	pendingMsgs []sdk.Msg
}

// provisionedOffChainActor is a struct containing the account and off-chain information
// of a provisioned actor.
type provisionedOffChainActor struct {
	accountInfo
	exposedServerAddress string
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

	s.blocksReplayObs = s.blockClient.CommittedBlocksSequence(s.ctx)

	// Initialize the funding account.
	s.initFundingAccount(fundingAccountKeyName)

	// TODO_IN_THIS_PR: source gateway config content
	s.provisionedGateways = []*provisionedOffChainActor{
		{accountInfo: accountInfo{keyName: `gateway1`}, exposedServerAddress: `http://localhost:42079`},
		{accountInfo: accountInfo{keyName: `gateway2`}, exposedServerAddress: `http://localhost:42080`},
		{accountInfo: accountInfo{keyName: `gateway3`}, exposedServerAddress: `http://localhost:42081`},
	}

	// TODO_IN_THIS_PR: source supplier config content
	s.provisionedSuppliers = []*provisionedOffChainActor{
		{accountInfo: accountInfo{keyName: `supplier1`}, exposedServerAddress: `http://relayminer1:8545`},
		{accountInfo: accountInfo{keyName: `supplier2`}, exposedServerAddress: `http://relayminer2:8545`},
		{accountInfo: accountInfo{keyName: `supplier3`}, exposedServerAddress: `http://relayminer3:8545`},
	}
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	// Create the initial suppliers
	supplierCount := table.Cell(3, 1).Int64()
	s.addInitialSuppliers(supplierCount)

	// Create the initial gateways
	gatewayCount := table.Cell(1, 1).Int64()
	s.addInitialGateways(gatewayCount)

	// Create the inital applications
	appCount := table.Cell(2, 1).Int64()
	s.addInitialApplications(appCount)

	// Fund all the initial actors
	s.sendFundInitialActorsMsgs(supplierCount, gatewayCount, appCount)
	s.waitForNextBlock()

	// Stake All the initial actors
	s.sendInitialActorsStakeMsgs(supplierCount, gatewayCount, appCount)
	s.waitForNextBlock()

	// Delegate all the initial applications to the initial gateways
	s.sendInitialDelegateMsgs(appCount, gatewayCount)
	s.waitForNextBlock()
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
	testDurationBlocks := math.Max(
		maxGateways/gatewayInc*gatewayBlockIncRate,
		maxApps/appInc*appBlockIncRate,
		maxSuppliers/supplierInc*supplierBlockIncRate,
	)

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
				Msgf("progress: %d/%d", blockHeight-s.startBlockHeight, testDurationBlocks)
			if blockHeight >= s.startBlockHeight+testDurationBlocks {
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

			if notif.blockHeight == notif.sessionStartBlockHeight {
				logger.Debug().
					Int64("session_num", notif.sessionNumber).
					Msg("activating prepared actors")

				// Activate teh prepared actors and prune the prepared lists.

				s.activeApplications = append(s.activeApplications, s.preparedApplications...)
				s.preparedApplications = []*accountInfo{}

				s.activeGateways = append(s.activeGateways, s.preparedGateways...)
				s.preparedGateways = []*provisionedOffChainActor{}

				s.activeSuppliers = append(s.activeSuppliers, s.preparedSuppliers...)
				s.preparedSuppliers = []*provisionedOffChainActor{}
			}

			if s.shouldIncrementActor(notif, supplierBlockIncRate, supplierInc, maxSuppliers) {
				// Incrementing suppliers can run concurrently with other actors since
				// they are not dependent on each other.
				go s.goIncrementSuppliers(notif, supplierInc, maxSuppliers)
			}

			// Get newly staked applications and gateways to create delegation messages
			// for them.
			// Stake messages are sent but not yet committed.

			var newGateways []*provisionedOffChainActor
			if s.shouldIncrementActor(notif, gatewayBlockIncRate, gatewayInc, maxGateways) {
				newGateways = s.stakeGateways(notif, gatewayInc, maxGateways)
			}

			var newApps []*accountInfo
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

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentPerApplicationPerSecond(appRPS string) {
	relayRatePerApp, err := strconv.ParseInt(appRPS, 10, 32)
	require.NoError(s, err)

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
		relaysPerSec := len(s.activeApplications) * int(relayRatePerApp)
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
) (newGateways []*provisionedOffChainActor) {
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
		s.sendTx(gateway.keyName, gateway.pendingMsgs...)
		gateway.pendingMsgs = []sdk.Msg{}
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
) (newApps []*accountInfo) {
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
		s.generateFundApplicationMsg(app)
		newApps = append(newApps, app)
	}
	s.sendTx(s.fundingAccountInfo.keyName, s.fundingAccountInfo.pendingMsgs...)
	s.fundingAccountInfo.pendingMsgs = []sdk.Msg{}

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
	newApps []*accountInfo,
	newGateways []*provisionedOffChainActor,
) {
	for _, app := range s.activeApplications {
		for _, gateway := range newGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		s.sendTx(app.keyName, app.pendingMsgs...)
		app.pendingMsgs = []sdk.Msg{}
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
		s.sendTx(app.keyName, app.pendingMsgs...)
		app.pendingMsgs = []sdk.Msg{}
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

	var newSuppliers []*provisionedOffChainActor
	for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
		supplier := s.addSupplier(supplierCount + supplierIdx)
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(supplier.keyName, supplier.pendingMsgs...)
		supplier.pendingMsgs = []sdk.Msg{}
		newSuppliers = append(newSuppliers, supplier)
	}
	s.waitForNextBlock()
	s.preparedSuppliers = append(s.preparedSuppliers, newSuppliers...)
}
