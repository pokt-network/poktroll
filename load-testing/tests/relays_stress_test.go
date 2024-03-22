package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/sync2"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/x/session/keeper"
)

const (
	// TODO_BLOCKER: parameterize blocks per second
	blocksPerSecond       = 1
	localnetPoktrollWSURL = "ws://localhost:42069/websocket"
	// maxConcurrentBatchLimit is the maximum number of concurrent relay batches that can be sent.
	//
	// TODO_TECHDEBT: consider parameterizing for cases where all CPUs are not
	// available (e.g. localnet is running on the same hardware).
	maxConcurrentBatchLimit = 2
	progressBarWidth        = 80
	defaultClearLineWidth   = 120
)

var (
	//localnetAnvilURL   string
	localnetGatewayURL string
	// maxConcurrentRequestLimit is the maximum number of concurrent requests that can be made.
	// By default, it is set to the number of logical CPUs available to the process.
	maxConcurrentRequestLimit = runtime.GOMAXPROCS(0)
)

type relaysSuite struct {
	gocuke.TestingT
	ctx             context.Context
	cancelCtx       context.CancelFunc
	startTime       time.Time
	blockClient     client.BlockClient
	blocksReplayObs client.BlockReplayObservable

	shouldRelayBatchBlocksObs observable.Observable[*relayBatchNotif]

	relaysSent     atomic.Uint64
	relaysComplete atomic.Uint64

	startingBlockHeight int64
	gatewayCount        atomic.Int64
	appCount            atomic.Int64
	supplierCount       atomic.Int64
	relaysPerSec        atomic.Int64
	nextRelaysPerSec    chan int64

	totalExpectedRequests uint64

	onChainStateChangeStartCh    chan struct{}
	onChainStateChangeCompleteCh chan struct{}
}

//type incNotif struct {
//	prevValue int64
//	nextValue int64
//}

type sessionInfoNotif struct {
	blockHeight             int64
	sessionNumber           int64
	sessionStartBlockHeight int64
	sessionEndBlockHeight   int64
}

type blockUpdatesChainStateNotif struct {
	//complete bool
	//done chan struct{}
	//blockHeight int64
	//testStep  int64
	sessionInfo  *sessionInfoNotif
	waitGroup    *sync.WaitGroup
	incGateways  bool
	incApps      bool
	incSuppliers bool
}

type relayBatchNotif struct {
	sessionInfo  *sessionInfoNotif
	batchNumber  int64
	relaysPerSec int64
}

func TestLoadRelays(t *testing.T) {
	gocuke.NewRunner(t, &relaysSuite{}).Path(filepath.Join(".", "relays_stress.feature")).Run()
}

func (s *relaysSuite) LocalnetIsRunning() {
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())

	// Cancel the context if this process is interrupted or exits.
	signals.GoOnExitSignal(func() {
		fmt.Println("")
		s.cancelCtx()
	})

	// TODO_TECHDEBT: add support for non-localnet environments.
	localnetGatewayURL = "http://localhost:42069/anvil"

	// Set up the block client.
	s.blockClient = testblock.NewLocalnetClient(s.ctx, s)
	//blockClientCtx, cancelBlocksObs := context.WithCancel(s.ctx)
	s.blocksReplayObs = s.blockClient.CommittedBlocksSequence(s.ctx)

	//s.blocksReplayObs = s.blockClient.CommittedBlocksSequence(blockClientCtx)
	//s.Cleanup(func() { cancelBlocksObs() })
	//s.onChainStateChangeStartCh = make(chan struct{}, 1)
	//s.onChainStateChangeCompleteCh = make(chan struct{}, 1)
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	// TODO_IN_THIS_COMMIT: account the difference between initially staked actors in
	// configured network and the target numbers initial actors.

	// Stake initial gateway(s)
	s.gatewayCount.Store(1)

	// Stake initial application(s)
	s.appCount.Store(1)

	// Stake initial supplier(s)
	s.supplierCount.Store(1)

}

func (s *relaysSuite) MoreActorsAreStakedAsFollows(table gocuke.DataTable) {
	gatewayInc := table.Cell(1, 1).Int64()
	gatewayBlockIncRate := table.Cell(1, 2).Int64()
	maxGateways := table.Cell(1, 3).Int64()
	require.Truef(s, gatewayBlockIncRate%keeper.NumBlocksPerSession == 0, "gateway increment rate must be a multiple of the session length")

	appInc := table.Cell(2, 1).Int64()
	appBlockIncRate := table.Cell(2, 2).Int64()
	maxApps := table.Cell(2, 3).Int64()
	require.Truef(s, appBlockIncRate%keeper.NumBlocksPerSession == 0, "app increment rate must be a multiple of the session length")

	supplierInc := table.Cell(3, 1).Int64()
	supplierBlockIncRate := table.Cell(3, 2).Int64()
	maxSuppliers := table.Cell(3, 3).Int64()
	require.Truef(s, supplierBlockIncRate%keeper.NumBlocksPerSession == 0, "supplier increment rate must be a multiple of the session length")

	batchNumber := new(atomic.Int64)
	// Start at batch number -1 ... TODO: explain why - modulus operator
	batchNumber.Add(-1)

	waitingForFirstSession := new(atomic.Bool)
	waitingForFirstSession.Store(true)

	// sessionInfoObs maps committed blocks to a notification which includes the
	// session number and the start and end block heights of the session.
	// It runs at the same frequency as committed blocks (i.e. 1:1).
	sessionInfoObs := channel.Map(s.ctx, s.blocksReplayObs,
		func(
			ctx context.Context,
			block client.Block,
		) (*sessionInfoNotif, bool) {
			sessionNum := block.Height() / keeper.NumBlocksPerSession
			sessionBlockIdx := block.Height() % keeper.NumBlocksPerSession
			sessionBlocksRemaining := keeper.NumBlocksPerSession - sessionBlockIdx

			// If the current block is not the first block of the session, wait for the
			// next session to start.
			if waitingForFirstSession.Load() && sessionBlockIdx != 0 {
				clearLine(s)
				logger.Info().
					Int64("block_height", block.Height()).
					Int64("session_num", sessionNum).
					Msgf("waiting for next session to start: in %d blocks", sessionBlocksRemaining)

				return nil, true
			}
			waitingForFirstSession.CompareAndSwap(true, false)

			// TODO_IN_THIS_COMMIT: check for off-by-one errors.
			return &sessionInfoNotif{
				blockHeight:             block.Height(),
				sessionNumber:           sessionNum,
				sessionStartBlockHeight: sessionNum * keeper.NumBlocksPerSession,
				sessionEndBlockHeight:   (sessionNum + 1) * keeper.NumBlocksPerSession,
			}, false
		},
	)

	// shouldBlockUpdateChainStateObs is an observable which is notified each block.
	// If the current "test height" is a multiple of any actor increment block count,
	// it ... TODO: finish
	shouldBlockUpdateChainStateObs := channel.Map(s.ctx, sessionInfoObs,
		func(
			ctx context.Context,
			sessionInfo *sessionInfoNotif,
		) (notif *blockUpdatesChainStateNotif, skip bool) {
			defer s.printProgressLine()

			// On the first block of each session, check if any new actors need to
			// be staked **for use in the next session**.
			// NB: assumes that the increment rates are multiples of the session length.
			// Otherwise, we would need to check if any block in the next session
			// is an increment height.

			nextSessionHeight := sessionInfo.sessionEndBlockHeight + 1

			// TODO_TECHDEBT(#21): replace with gov param query when available.
			gatewaySessionIncRate := gatewayBlockIncRate / keeper.NumBlocksPerSession
			isGatewayStakeHeight := nextSessionHeight%(gatewaySessionIncRate) == 0

			// TODO_TECHDEBT(#21): replace with gov param query when available.
			appSessionIncRate := appBlockIncRate / keeper.NumBlocksPerSession
			isAppStakeHeight := nextSessionHeight%(appSessionIncRate) == 0

			// TODO_TECHDEBT(#21): replace with gov param query when available.
			supplierSessionIncRate := supplierBlockIncRate / keeper.NumBlocksPerSession
			isSupplierStakeHeight := nextSessionHeight%(supplierSessionIncRate) == 0

			isSessionStartHeight := sessionInfo.blockHeight == sessionInfo.sessionStartBlockHeight

			// If the current height is not a session start or an actor increment
			// height, notify downstream observables but omit the wait group.
			if !isSessionStartHeight ||
				!isGatewayStakeHeight &&
					!isAppStakeHeight &&
					!isSupplierStakeHeight {
				clearLine(s)
				logger.Debug().
					Int64("block_height", sessionInfo.blockHeight).
					Int64("session_num", sessionInfo.sessionNumber).
					Msg("no chain state updates required")

				return &blockUpdatesChainStateNotif{
					sessionInfo: sessionInfo,
					// waitGroup explicitly omitted to signal no async updates.
				}, false
			}

			clearLine(s)
			logger.Debug().
				Int64("block_height", sessionInfo.blockHeight).
				Int64("session_num", sessionInfo.sessionNumber).
				Msg("actor stake height detected")

			// This test step requires chain state updates, include a wait group
			// for use by downstream observables.
			notif = &blockUpdatesChainStateNotif{
				sessionInfo:  sessionInfo,
				waitGroup:    &sync.WaitGroup{},
				incGateways:  isGatewayStakeHeight,
				incApps:      isAppStakeHeight,
				incSuppliers: isSupplierStakeHeight,
			}

			return notif, false

		},
	)

	//isChainStateUpdating := new(atomic.Bool)

	blockUpdatesChainStateObs := channel.Map(s.ctx, shouldBlockUpdateChainStateObs,
		func(ctx context.Context, notif *blockUpdatesChainStateNotif) (*blockUpdatesChainStateNotif, bool) {
			defer s.printProgressLine()

			sessionInfo := notif.sessionInfo

			// If the notification wait group is nil there is no update to the chain state.
			if notif.waitGroup == nil {
				return notif, false
			}

			if notif.incGateways {
				s.incrementGateways(notif, gatewayInc, maxGateways)
			}

			if notif.incApps {
				s.incrementApps(notif, appInc, maxApps)
			}

			if notif.incSuppliers {
				s.incrementSuppliers(notif, supplierInc, maxSuppliers)
			}

			clearLine(s)
			logger.Debug().
				Int64("block_height", sessionInfo.blockHeight).
				Int64("session_num", sessionInfo.sessionNumber).
				Msg("waiting for chain state updates")
			notif.waitGroup.Wait()

			// Increment the test step after the chain state updates are complete.
			clearLine(s)
			logger.Debug().
				Int64("block_height", sessionInfo.blockHeight).
				Int64("session_num", sessionInfo.sessionNumber).
				Msg("chain state updates complete")

			return notif, false
		},
	)

	// TODO_IN_THIS_COMMIT: consider moving to #ALoadOfConcurrentRelayRequestsAreSent
	s.shouldRelayBatchBlocksObs = channel.Map(s.ctx, blockUpdatesChainStateObs,
		func(ctx context.Context, notif *blockUpdatesChainStateNotif) (*relayBatchNotif, bool) {
			defer s.printProgressLine()

			// Increment the batch number.
			nextBatchNumber := batchNumber.Add(1)

			clearLine(s)
			logger.Debug().
				Int64("block_height", notif.sessionInfo.blockHeight).
				Int64("session_num", notif.sessionInfo.sessionNumber).
				Int64("prev_batch_number", nextBatchNumber-1).
				Int64("next_batch_number", nextBatchNumber).
				Msg("incrementing batch number")
			//testStep.Add(1)

			return &relayBatchNotif{
				sessionInfo: notif.sessionInfo,
				batchNumber: nextBatchNumber,
			}, false
		},
	)
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentPerSecondAsFollows(table gocuke.DataTable) {
	initialRelayRate := table.Cell(1, 0).Int64()
	s.relaysPerSec.Store(initialRelayRate)

	relaysRateInc := table.Cell(1, 1).Int64()
	relayRateBlocksInc := table.Cell(1, 2).Int64()
	maxRelaysRate := table.Cell(1, 3).Int64()

	// Set the total number of relay requests to be sent.
	// It may be read from concurrently running goroutines but remains
	// constant for the duration of the test.
	s.totalExpectedRequests = computeTotalRequests(
		initialRelayRate,
		relaysRateInc,
		relayRateBlocksInc,
		maxRelaysRate,
	)

	// relayBatchObs maps from block heights at which a relay batch should be sent to
	// the number of relays per second to send in that batch, incrementing the rps
	// according to the step table.
	relayBatchObs := channel.Map(s.ctx, s.shouldRelayBatchBlocksObs,
		func(ctx context.Context, notif *relayBatchNotif) (*relayBatchNotif, bool) {
			relaysPerSec := s.relaysPerSec.Load()

			if notif.batchNumber != 0 &&
				notif.batchNumber%relayRateBlocksInc == 0 {
				// Increment relaysPerSec.
				relaysPerSec = s.relaysPerSec.Add(relaysRateInc)
			}

			// Populate the number of relay requests to send in this batch.
			notif.relaysPerSec = relaysPerSec

			return notif, false
		},
	)

	// tickerCircuitBreaker is used to limit the concurrency of batches and error
	// if the limit would be exceeded.
	tickerCircuitBreaker := sync2.NewCircuitBreaker(maxConcurrentBatchLimit)
	// batchLimiter limits request concurrency to match the maximum supported by hardware.
	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)

	channel.ForEach(s.ctx, relayBatchObs,
		func(ctx context.Context, batch *relayBatchNotif) {
			relayRate := batch.relaysPerSec
			batchWaitGroup := sync.WaitGroup{}

			// Send relay batch...
			tickerCircuitBreaker.Go(s.ctx, func() {
				remainingRelays := s.totalExpectedRequests - s.relaysComplete.Load()
				// Ensure the number of relays sent in this batch does not exceed the maximum.
				// I.e. this is the last batch.
				if remainingRelays < uint64(relayRate) {
					relayRate = int64(remainingRelays)
				}

				batchWaitGroup.Add(int(relayRate))
				for i := int64(0); i < relayRate; i++ {
					// Abort remaining relays in this batch if the context was cancelled.
					select {
					case <-s.ctx.Done():
						return
					default:
					}

					// Each relay should not block on any other relay; however,
					// maximum concurrency is limited by hardware capabilities.
					batchLimiter.Go(s.ctx, func() {
						s.relaysSent.Add(1)

						// Permute & distribute relays across all applications and gateways...

						// Send relay...
						// TODO: resume here!!!
						// TODO: resume here!!!
						// TODO: resume here!!!
						time.Sleep(time.Millisecond * 250)

						s.relaysComplete.Add(1)

						batchWaitGroup.Done()
					})
				}

				// relayRate remains at maxRelayRate for relayRateBlocksInc blocks worth of batches.
				if relayRate == maxRelaysRate &&
					batch.batchNumber%relayRateBlocksInc == relayRateBlocksInc-1 {
					batchWaitGroup.Wait()
					s.printProgressLine()
					s.cancelCtx()
				}

			})

			// Wait for the batch asynchronously to avoid creating backpressure in
			// this observable such that the circuit breaker becomes ineffective.
			go func() {
				defer s.printProgressLine()

				batchWaitGroup.Wait()

				clearLine(s)
				logger.Info().
					Int64("session_num", batch.sessionInfo.sessionNumber).
					Int64("block_height", batch.sessionInfo.blockHeight).
					Int64("batch_number", batch.batchNumber).
					Msgf(
						"batch %d complete (%d/%d)",
						batch.batchNumber,
						relayRate,
						relayRate,
					)
			}()
		},
	)

	// Wait for the suite context to be done.
	<-s.ctx.Done()
}

func (s *relaysSuite) incrementGateways(
	notif *blockUpdatesChainStateNotif,
	gatewayInc,
	maxGateways int64,
) {
	gatewayCount := s.gatewayCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if gatewayCount == maxGateways {
		clearLine(s)
		logger.Debug().
			Int64("block_height", notif.sessionInfo.blockHeight).
			Int64("session_num", notif.sessionInfo.sessionNumber).
			Msg("skipping gateway increment, max gateways reached")

		return
	}

	gatewaysToStake := gatewayInc
	if gatewayCount+gatewaysToStake > maxGateways {
		gatewaysToStake = maxGateways - gatewayCount
	}

	notif.waitGroup.Add(int(gatewaysToStake))

	go func() {
		defer s.printProgressLine()

		// Stake gateways...
		clearLine(s)
		logger.Info().Msgf(
			"staking gateways for session %d (%d->%d)",
			notif.sessionInfo.sessionNumber+1,
			gatewayCount,
			gatewayCount+gatewaysToStake,
		)

		for gwIdx := int64(0); gwIdx < gatewaysToStake; gwIdx++ {
			go func() {
				time.Sleep(250)

				s.gatewayCount.Add(1)
				notif.waitGroup.Done()
			}()
		}
	}()
}

func (s *relaysSuite) incrementApps(
	notif *blockUpdatesChainStateNotif,
	appIncAmt,
	maxApps int64,
) {
	appCount := s.appCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if appCount == maxApps {
		clearLine(s)
		logger.Debug().
			Int64("block_height", notif.sessionInfo.blockHeight).
			Int64("session_num", notif.sessionInfo.sessionNumber).
			Msg("skipping app increment, max apps reached")

		return
	}

	appsToStake := appIncAmt
	if appCount+appsToStake > maxApps {
		appsToStake = maxApps - appCount
	}

	notif.waitGroup.Add(int(appsToStake))

	go func() {
		defer s.printProgressLine()

		// Stake applications...
		clearLine(s)
		logger.Info().Msgf(
			"staking applications for session %d (%d->%d)",
			notif.sessionInfo.sessionNumber+1,
			appCount,
			appCount+appsToStake,
		)

		for appIdx := int64(0); appIdx < appsToStake; appIdx++ {
			go func() {
				time.Sleep(250)

				s.appCount.Add(1)
				notif.waitGroup.Done()
			}()
		}
	}()
}

func (s *relaysSuite) incrementSuppliers(
	notif *blockUpdatesChainStateNotif,
	supplierInc,
	maxSuppliers int64,
) {
	supplierCount := s.supplierCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if supplierCount == maxSuppliers {
		clearLine(s)
		logger.Debug().
			Int64("block_height", notif.sessionInfo.blockHeight).
			Int64("session_num", notif.sessionInfo.sessionNumber).
			Msg("skipping supplier increment, max suppliers reached")

		return
	}

	suppliersToStake := supplierInc
	if supplierCount+suppliersToStake > maxSuppliers {
		suppliersToStake = maxSuppliers - supplierCount
	}

	notif.waitGroup.Add(int(suppliersToStake))

	go func() {
		defer s.printProgressLine()

		// Stake suppliers...
		clearLine(s)
		logger.Info().Msgf(
			"staking suppliers for session %d (%d->%d)",
			notif.sessionInfo.sessionNumber+1,
			supplierCount,
			supplierCount+suppliersToStake,
		)

		for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
			go func() {
				time.Sleep(250)

				s.supplierCount.Add(1)
				notif.waitGroup.Done()
			}()
		}
	}()
}

// clearLine clears the current line in the console.
func clearLine(t gocuke.TestingT) {
	t.Helper()

	fmt.Printf("\r%s", strings.Repeat(" ", getTermWidth(t)))
	fmt.Print("\r")
}

// printProgressLine prints a progress bar to the console.
func (s *relaysSuite) printProgressLine() {
	s.Helper()

	completeCount := s.relaysComplete.Load()
	totalCount := s.totalExpectedRequests

	var completeChars, pendingChars uint64

	if totalCount != 0 {
		completeChars = progressBarWidth * completeCount / totalCount
		pendingChars = progressBarWidth - completeChars
	}

	if pendingChars+completeChars > progressBarWidth {
		clearLine(s)
		logger.Warn().Msg("progress bar overflowed")
	}

	// Print the progress bar
	fmt.Printf(
		"\r[%s%s] (%d/%d)",
		//"\n[%s%s] (%d/%d)",
		strings.Repeat("=", int(completeChars)),
		strings.Repeat(" ", int(pendingChars)),
		completeCount,
		totalCount,
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

// computeTotalRequests calculates the total number of relay requests to be sent
// by integrating the number of relays per second over time.
func computeTotalRequests(initialRelaysRate, relayRateInc, relayRateBlocksInc, maxRelaysRate int64) uint64 {
	var totalRequests uint64
	for relayRate := initialRelaysRate; relayRate <= maxRelaysRate; relayRate += relayRateInc {
		totalRequests += uint64(relayRate * relayRateBlocksInc * blocksPerSecond)
	}
	return totalRequests
}
