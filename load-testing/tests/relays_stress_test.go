package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"

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

const (
	// TODO_TECHDEBT: consider parameterizing for cases where all CPUs are not
	// available (e.g. localnet is running on the same hardware).
	maxConcurrentBatchLimit = 2
	progressBarWidth        = 80
	defaultClearLineWidth   = 120
)

var (
	// maxConcurrentRequestLimit is the maximum number of concurrent requests that can be made.
	// By default, it is set to the number of logical CPUs available to the process.
	maxConcurrentRequestLimit = runtime.GOMAXPROCS(0)
	fundingAccountKeyName     = "pnf"
	fundingAmount             = sdk.NewCoin("upokt", math.NewInt(10000000))
	stakeAmount               = sdk.NewCoin("upokt", math.NewInt(10000))
	applicationStakeAmount    = sdk.NewCoin("upokt", math.NewInt(100000))
	anvilService              = &sharedtypes.Service{Id: "anvil"}
)

type relaysSuite struct {
	gocuke.TestingT
	ctx             context.Context
	cancelCtx       context.CancelFunc
	blockClient     client.BlockClient
	txContext       client.TxContext
	blocksReplayObs client.BlockReplayObservable

	sessionInfoObs observable.Observable[*sessionInfoNotif]

	fundingAccountInfo *accountInfo
	relaysSent         atomic.Uint64

	waitingForFirstSession atomic.Bool

	provisionedGateways  []*provisionedOffChainActor
	provisionedSuppliers []*provisionedOffChainActor

	preparedGateways     []*provisionedOffChainActor
	preparedSuppliers    []*provisionedOffChainActor
	preparedApplications []*accountInfo

	activeGateways     []*provisionedOffChainActor
	activeSuppliers    []*provisionedOffChainActor
	activeApplications []*accountInfo
}

type accountInfo struct {
	keyName     string
	accAddress  sdk.AccAddress
	privKey     *secp256k1.PrivKey
	pendingMsgs []sdk.Msg
}

type provisionedOffChainActor struct {
	accountInfo
	exposedServerAddress string
}

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
	s.blockClient = testblock.NewLocalnetClient(s.ctx, s)

	// Setup the txClient
	s.txContext = testtx.NewLocalnetContext(s.TestingT.(*testing.T))

	s.blocksReplayObs = s.blockClient.CommittedBlocksSequence(s.ctx)

	s.initFundingAccount(fundingAccountKeyName)

	// TODO_IN_THIS_COMMIT: source gateway config content
	s.provisionedGateways = []*provisionedOffChainActor{
		{accountInfo: accountInfo{keyName: `gateway1`}, exposedServerAddress: `http://localhost:42079`},
		{accountInfo: accountInfo{keyName: `gateway2`}, exposedServerAddress: `http://localhost:42080`},
		{accountInfo: accountInfo{keyName: `gateway3`}, exposedServerAddress: `http://localhost:42081`},
	}

	// TODO_IN_THIS_COMMIT: source supplier config content
	s.provisionedSuppliers = []*provisionedOffChainActor{
		{accountInfo: accountInfo{keyName: `supplier1`}, exposedServerAddress: `http://relayminer1:8545`},
		{accountInfo: accountInfo{keyName: `supplier2`}, exposedServerAddress: `http://relayminer2:8545`},
		{accountInfo: accountInfo{keyName: `supplier3`}, exposedServerAddress: `http://relayminer3:8545`},
	}
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	supplierCount := table.Cell(3, 1).Int64()
	s.addInitialSuppliers(supplierCount)

	gatewayCount := table.Cell(1, 1).Int64()
	s.addInitialGateways(gatewayCount)

	appCount := table.Cell(2, 1).Int64()
	s.addInitialApplications(appCount)

	s.sendFundInitialActorsMsgs(supplierCount, gatewayCount, appCount)
	s.waitForNextBlock()

	s.sendInitialActorsStakeMsgs(supplierCount, gatewayCount, appCount)
	s.waitForNextBlock()

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

	s.waitingForFirstSession.Store(true)

	testDurationBlocks := math.Max(
		maxGateways/gatewayInc*gatewayBlockIncRate,
		maxApps/appInc*appBlockIncRate,
		maxSuppliers/supplierInc*supplierBlockIncRate,
	)

	var startBlockHeight int64

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

			sessionBlocksRemaining := sessionInfo.sessionEndBlockHeight - sessionInfo.blockHeight

			// If the current block is not the first block of the session, wait for the
			// next session to start.
			if s.waitingForFirstSession.Load() && sessionInfo.blockHeight != sessionInfo.sessionStartBlockHeight {
				clearLine(s)
				logger.Info().
					Int64("block_height", block.Height()).
					Int64("session_num", sessionInfo.sessionNumber).
					Msgf("waiting for next session to start: in %d blocks", sessionBlocksRemaining)

				return nil, true
			}

			if s.waitingForFirstSession.Load() {
				startBlockHeight = blockHeight
			}

			s.waitingForFirstSession.CompareAndSwap(true, false)

			// [Active][Gw: 1/3, Supp: 1/3, App: 1/10]
			// [Prepared][Gw: 1, Supp: 1, App: 1]
			// [Relays][RPS: 10, Sent: 100]
			// [Session][Num: 1, Start: 5, End, 9]
			s.printProgressLine(blockHeight-startBlockHeight, testDurationBlocks)
			if blockHeight >= startBlockHeight+testDurationBlocks {
				s.cancelCtx()
			}

			return sessionInfo, false
		},
	)

	// shouldBlockUpdateChainStateObs is an observable which is notified each block.
	// If the current "test height" is a multiple of any actor increment block count.
	channel.ForEach(s.ctx, s.sessionInfoObs,
		func(ctx context.Context, notif *sessionInfoNotif) {
			// On the first block of each session, check if any new actors need to
			// be staked **for use in the next session**.
			// NB: assumes that the increment rates are multiples of the session length.
			// Otherwise, we would need to check if any block in the next session
			// is an increment height.

			if notif.blockHeight == notif.sessionStartBlockHeight {
				s.activeApplications = append(s.activeApplications, s.preparedApplications...)
				s.preparedApplications = []*accountInfo{}

				s.activeGateways = append(s.activeGateways, s.preparedGateways...)
				s.preparedGateways = []*provisionedOffChainActor{}

				s.activeSuppliers = append(s.activeSuppliers, s.preparedSuppliers...)
				s.preparedSuppliers = []*provisionedOffChainActor{}
			}

			if s.shouldIncrementActor(notif, supplierBlockIncRate, supplierInc, maxSuppliers) {
				go s.goIncrementSuppliers(notif, supplierInc, maxSuppliers)
			}

			var newGateways []*provisionedOffChainActor
			if s.shouldIncrementActor(notif, gatewayBlockIncRate, gatewayInc, maxGateways) {
				newGateways = s.stakeGateways(notif, gatewayInc, maxGateways)
			}

			var newApps []*accountInfo
			if s.shouldIncrementActor(notif, appBlockIncRate, appInc, maxApps) {
				newApps = s.stakeApps(notif, appInc, maxApps)
			}

			s.waitForNextBlock()
			s.stakeAndDelegateApps(newApps, newGateways)
		},
	)
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentPerApplicationPerSecond(appRPS string) {
	relayRatePerApp, err := strconv.ParseInt(appRPS, 10, 32)
	require.NoError(s, err)

	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if s.waitingForFirstSession.Load() {
			continue
		}

		batchLimiter.Go(s.ctx, func() {
			s.sendRelay(s.relaysSent.Add(1) - 1)
		})

		relaysPerSec := len(s.activeApplications) * int(relayRatePerApp)
		relayInterval := time.Second / time.Duration(relaysPerSec)
		time.Sleep(relayInterval)
	}
}

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

	// Stake gateways...
	clearLine(s)
	logger.Info().Msgf(
		"staking gateways for session %d (%d->%d)",
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

	return newGateways
}

func (s *relaysSuite) stakeApps(
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

	clearLine(s)
	logger.Info().Msgf(
		"staking applications for session %d (%d->%d)",
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

	return newApps
}

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

	s.waitForNextBlock()
	s.preparedGateways = append(s.preparedGateways, newGateways...)
	s.preparedApplications = append(s.preparedApplications, newApps...)
}

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

	clearLine(s)
	logger.Info().Msgf(
		"staking suppliers for session %d (%d->%d)",
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

// printProgressLine prints a progress bar to the console.
func (s *relaysSuite) printProgressLine(blocksProgress, testDurationBlocks int64) {
	s.Helper()

	var completeChars, pendingChars int64

	if testDurationBlocks != 0 {
		completeChars = progressBarWidth * blocksProgress / testDurationBlocks
		pendingChars = progressBarWidth - completeChars
	}

	if completeChars > progressBarWidth {
		completeChars = progressBarWidth
	}

	if pendingChars < 0 {
		pendingChars = 0
	}

	// Print the progress bar
	fmt.Printf(
		"\r[%s%s] (%d/%d)",
		//"\n[%s%s] (%d/%d)",
		strings.Repeat("=", int(completeChars)),
		strings.Repeat(" ", int(pendingChars)),
		blocksProgress,
		testDurationBlocks,
	)
}

// getTermWidth returns the width of the terminal. If the width cannot be
// determined, it returns a default width.
func getTermWidth(t gocuke.TestingT) int {
	t.Helper()

	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = defaultClearLineWidth
	}

	return width
}

// clearLine clears the current line in the console.
func clearLine(t gocuke.TestingT) {
	t.Helper()

	fmt.Printf("\r%s", strings.Repeat(" ", getTermWidth(t)))
	fmt.Print("\r")
}
