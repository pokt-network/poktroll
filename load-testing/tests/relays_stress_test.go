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

type blockUpdatesChainStateNotif struct {
	//complete bool
	//done chan struct{}
	//blockHeight int64
	testStep  int64
	waitGroup *sync.WaitGroup
}

type relayBatchNotif struct {
	relaysPerSec int64
	batchNumber  int64
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
	gatewayIncRate := table.Cell(1, 2).Int64()
	maxGateways := table.Cell(1, 3).Int64()

	appInc := table.Cell(2, 1).Int64()
	appIncRate := table.Cell(2, 2).Int64()
	maxApps := table.Cell(2, 3).Int64()

	supplierInc := table.Cell(3, 1).Int64()
	supplierIncRate := table.Cell(3, 2).Int64()
	maxSuppliers := table.Cell(3, 3).Int64()

	// TODO_IN_THIS_COMMIT: consider moving initialization to #LocalnetIsRunning and
	// making it a field of the suite.
	testStep := new(atomic.Int64)
	// Start at test stap 1 ... TODO: explain why - modulus operator
	testStep.Add(1)

	batchNumber := new(atomic.Int64)
	// Start at batch number -1 ... TODO: explain why - modulus operator
	batchNumber.Add(-1)

	//batchObs := channel.Map(s.ctx, s.blocksReplayObs,
	//	func(ctx context.Context, block client.Block) (*atomic.Uint64, bool) {
	//		// Increment the batch number every time a block is received.
	//		testStep.Add(1)
	//
	//		return testStep, false
	//	},
	//)

	// shouldBlockUpdateChainStateObs is an observable which is notified each block.
	// If the current "test height" is a multiple of any actor increment block count,
	// it ... TODO: finish
	shouldBlockUpdateChainStateObs := channel.Map(s.ctx, s.blocksReplayObs,
		func(
			ctx context.Context,
			block client.Block,
		) (notif *blockUpdatesChainStateNotif, skip bool) {
			testStep := testStep.Load()
			//logger.Debug().
			//	Int64("block_height", block.Height()).
			//	Int64("test_height", testStep).
			//	Msg("new block")

			isGatewayIncStartStep := testStep%(gatewayIncRate+2) == gatewayIncRate+1
			isGatewayIncEndStep := testStep%(gatewayIncRate+2) == 0
			isAppIncStartStep := testStep%(appIncRate+2) == appIncRate+1
			isAppIncEndStep := testStep%(appIncRate+2) == 0
			isSupplierIncStartStep := testStep%(supplierIncRate+2) == supplierIncRate+1
			isSupplierIncEndStep := testStep%(supplierIncRate+2) == 0

			// If the test step is not a chain state update step, notify
			// downstream observables but without a wait group.
			if isGatewayIncStartStep ||
				isAppIncStartStep ||
				isSupplierIncStartStep {

				clearLine(s)
				logger.Debug().
					Int64("block_height", block.Height()).
					Int64("testStep", testStep).
					Msg("chain state updates required")

				// This test step requires chain state updates, include a wait group
				// for use by downstream observables.
				notif = &blockUpdatesChainStateNotif{
					testStep:  testStep,
					waitGroup: &sync.WaitGroup{},
				}

				return notif, false
			}

			if isGatewayIncEndStep ||
				isAppIncEndStep ||
				isSupplierIncEndStep {
				// TODO: is this correct?
				//return nil, true
				return nil, false
				//return &blockUpdatesChainStateNotif{
				//	testStep: testStep,
				//}
			}

			clearLine(s)
			logger.Debug().
				Int64("block_height", block.Height()).
				Int64("testStep", testStep).
				Msg("no chain state updates")

			// There are no new actors to stake in the given block.
			// Return nil to indicate no state updates need to be made
			// and don't skip in order to signal that a block has ticked.
			return &blockUpdatesChainStateNotif{
				testStep: testStep,
				// waitGroup explicitly set to nil to indicate no state updates
			}, false
		},
	)

	isChainStateUpdating := new(atomic.Bool)

	blockUpdatesChainStateObs := channel.Map(s.ctx, shouldBlockUpdateChainStateObs,
		func(ctx context.Context, notif *blockUpdatesChainStateNotif) (*blockUpdatesChainStateNotif, bool) {
			if notif == nil {
				testStep.Add(1)
				return nil, true
			}

			// If the notification wait group is nil there is no update to the chain state.
			if notif.waitGroup == nil {
				// Increment the test step.
				clearLine(s)
				logger.Debug().
					Int64("testStep", notif.testStep).
					Msg("incrementing test step only")
				testStep.Add(1)

				return notif, false
			}

			clearLine(s)
			logger.Debug().
				Int64("testStep", notif.testStep).
				Msg("marking chain state as updating")
			isNotAlreadyUpdating := isChainStateUpdating.CompareAndSwap(false, true)
			require.Truef(s, isNotAlreadyUpdating, "chain state attempted to change while previous change was still in progress")

			s.incrementGateways(notif, gatewayIncRate, gatewayInc, maxGateways)
			s.incrementApps(notif, appIncRate, appInc, maxApps)
			s.incrementSuppliers(notif, supplierIncRate, supplierInc, maxSuppliers)

			// TODO_IN_THIS_COMMIT: something more elegant than sleeping
			// Wait a tick for the wait group to be incremented to before
			// waiting on it.
			time.Sleep(100 * time.Millisecond)

			go func() {
				clearLine(s)
				logger.Debug().
					Int64("testStep", notif.testStep).
					Msg("waiting for chain state updates")
				notif.waitGroup.Wait()

				clearLine(s)
				logger.Debug().
					Int64("testStep", notif.testStep).
					Msg("marking chain state update complete")
				isChainStateUpdating.CompareAndSwap(true, false)

				// Increment the test step after the chain state updates are complete.
				clearLine(s)
				logger.Debug().
					Int64("testStep", notif.testStep).
					Msg("incrementing test step only")
				testStep.Add(1)
			}()

			return notif, false
		},
	)

	// TODO_IN_THIS_COMMIT: consider moving to #ALoadOfConcurrentRelayRequestsAreSent
	s.shouldRelayBatchBlocksObs = channel.Map(s.ctx, blockUpdatesChainStateObs,
		func(ctx context.Context, notif *blockUpdatesChainStateNotif) (*relayBatchNotif, bool) {
			//// If there are chain state updates, wait for them to complete first.
			//if notif.waitGroup != nil {
			//}

			// If the chain state is updating, skip the batch(es) that would
			// otherwise be scheduled for this block. I.e. do NOT increment
			// the test height.
			if isChainStateUpdating.Load() {
				logger.Debug().
					Int64("testStep", notif.testStep).
					Int64("batch_number", batchNumber.Load()).
					Msg("shouldRelayBatchBlocksObs skipping; chain state updating")
				return nil, true
			}

			// Increment test height for each block where no chain state
			// updates are in progress.
			//logger.Debug().Msg("incrementing test step & batch number")
			//testStep.Add(1)
			nextBatchNumber := batchNumber.Add(1)

			logger.Debug().
				Int64("testStep", notif.testStep).
				Int64("prev_batch_number", nextBatchNumber-1).
				Int64("next_batch_number", nextBatchNumber).
				Msg("incrementing batch number only")
			//testStep.Add(1)

			return &relayBatchNotif{
				batchNumber: nextBatchNumber,
			}, false
		},
	)
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentPerSecondAsFollows(table gocuke.DataTable) {
	// Set initial relays per second
	initialRelaysPerSecond := table.Cell(1, 0).Int64()
	s.relaysPerSec.Store(initialRelaysPerSecond)

	relaysPerSecInc := table.Cell(1, 1).Int64()
	numBlocksPerInc := table.Cell(1, 2).Int64()
	maxRelaysPerSec := table.Cell(1, 3).Int64()

	// Set the total number of relay requests to be sent.
	// It may be read from concurrently running goroutines but remains
	// constant for the duration of the test.
	s.totalExpectedRequests = computeTotalRequests(initialRelaysPerSecond, relaysPerSecInc, numBlocksPerInc, maxRelaysPerSec)

	// relayBatchObs maps from block heights at which a relay batch should be sent to
	// the number of relays per second to send in that batch, incrementing the rps
	// according to the step table.
	relayBatchObs := channel.Map(s.ctx, s.shouldRelayBatchBlocksObs,
		func(ctx context.Context, notif *relayBatchNotif) (*relayBatchNotif, bool) {
			relaysPerSec := s.relaysPerSec.Load()

			if notif.batchNumber != 0 &&
				notif.batchNumber%numBlocksPerInc == 0 {
				// Increment relaysPerSec.
				relaysPerSec = s.relaysPerSec.Add(relaysPerSecInc)
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
			relaysPerSec := batch.relaysPerSec
			batchWaitGroup := sync.WaitGroup{}

			// Send relay batch...
			tickerCircuitBreaker.Go(s.ctx, func() {
				remainingRelays := s.totalExpectedRequests - s.relaysComplete.Load()
				// Ensure the number of relays sent in this batch does not exceed the maximum.
				// I.e. this is the last batch.
				if remainingRelays < uint64(relaysPerSec) {
					relaysPerSec = int64(remainingRelays)
				}

				batchWaitGroup.Add(int(relaysPerSec))
				for i := int64(0); i < relaysPerSec; i++ {
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

			})

			// Wait for the batch asynchronously to avoid creating backpressure in
			// this observable such that the circuit breaker becomes ineffective.
			go func() {
				batchWaitGroup.Wait()

				clearLine(s)
				logger.Info().Msgf(
					"batch number %d complete (%d/%d)",
					batch.batchNumber,
					relaysPerSec,
					relaysPerSec,
				)
				printProgressLine(s, progressBarWidth, s.relaysComplete.Load(), s.totalExpectedRequests)
			}()
		},
	)

	// Wait for the suite context to be done.
	<-s.ctx.Done()
}

func (s *relaysSuite) incrementGateways(
	notif *blockUpdatesChainStateNotif,
	gatewayIncRate,
	gatewayInc,
	maxGateways int64,
) {
	// Return early if the test height is not a multiple of the
	// gateway increment block count.
	if notif.testStep%(gatewayIncRate+2) != gatewayIncRate+1 {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping gateway increment")

		return
	}

	gatewayCount := s.gatewayCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if gatewayCount == maxGateways {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping gateway increment, max gateways reached")

		return
	}

	gatewaysToStake := gatewayInc
	if gatewayCount+gatewaysToStake > maxGateways {
		gatewaysToStake = maxGateways - gatewayCount
	}

	notif.waitGroup.Add(int(gatewaysToStake))

	go func() {
		// Stake gateways...
		clearLine(s)
		logger.Info().Msgf(
			"incrementing staked gateways (%d->%d)",
			gatewayCount,
			gatewayCount+gatewaysToStake,
		)

		for gwIdx := int64(0); gwIdx < gatewaysToStake; gwIdx++ {
			time.Sleep(250)

			s.gatewayCount.Add(1)
			notif.waitGroup.Done()
		}
	}()
}

func (s *relaysSuite) incrementApps(
	notif *blockUpdatesChainStateNotif,
	appIncRate,
	appIncAmt,
	maxApps int64,
) {
	// Return early if the test height is not a multiple of the
	// gateway increment block count.
	if notif.testStep%(appIncRate+2) != appIncRate+1 {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping app increment")

		return
	}

	appCount := s.appCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if appCount == maxApps {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping app increment, max apps reached")

		return
	}

	appsToStake := appIncAmt
	if appCount+appsToStake > maxApps {
		appsToStake = maxApps - appCount
	}

	notif.waitGroup.Add(int(appsToStake))

	go func() {
		// Stake applications...
		clearLine(s)
		logger.Info().Msgf(
			"incrementing staked applications (%d->%d)",
			appCount,
			appCount+appsToStake,
		)

		for appIdx := int64(0); appIdx < appsToStake; appIdx++ {
			time.Sleep(250)

			s.appCount.Add(1)
			notif.waitGroup.Done()
		}
	}()
}

func (s *relaysSuite) incrementSuppliers(
	notif *blockUpdatesChainStateNotif,
	supplierIncRate,
	supplierInc,
	maxSuppliers int64,
) {
	// Return early if the test height is not a multiple of the
	// gateway increment block count.
	if notif.testStep%(supplierIncRate+2) != supplierIncRate+1 {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping supplier increment, not an stake supplier block")

		return
	}

	supplierCount := s.supplierCount.Load()

	// TODO_IN_THIS_COMMIT: move this check upstream in the pipeline
	// (e.g. into shouldBlockUpdateChainStateObs)
	if supplierCount == maxSuppliers {
		logger.Debug().
			Int64("testStep", notif.testStep).
			Msg("skipping supplier increment, max suppliers reached")

		return
	}

	suppliersToStake := supplierInc
	if supplierCount+suppliersToStake > maxSuppliers {
		suppliersToStake = maxSuppliers - supplierCount
	}

	notif.waitGroup.Add(int(suppliersToStake))

	go func() {
		// Stake suppliers...
		clearLine(s)
		logger.Info().Msgf(
			"incrementing staked suppliers (%d->%d)",
			supplierCount,
			supplierCount+suppliersToStake,
		)

		for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
			time.Sleep(250)

			s.supplierCount.Add(1)
			notif.waitGroup.Done()
		}
	}()
}

// goStartRelayBatchTicker starts a ticker that sends relay batches at a rate
// determined by the number of relays per second. It also limits the number of
// concurrent relay batches that can be sent. If the limit is exceeded, it will
// error and fail the test.
// It is intended to be run in a goroutine.
func (s *relaysSuite) goStartRelayBatchTicker(maxConcurrentBatchLimit uint, maxRelaysPerSec int64) {
	// Synchronize initial batch start with goIncRelaysPerSec (next block height)..
	blocksSubCtx, cancelBlocksSub := context.WithCancel(s.ctx)
	blocksCh := s.blocksReplayObs.Subscribe(blocksSubCtx).Ch()
	//<-s.blocksReplayObs.Subscribe(blocksSubCtx).Ch()
	for block := range blocksCh {
		if block.Height() > s.startingBlockHeight {
			break
		}
	}
	cancelBlocksSub()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// tickerCircuitBreaker is used to limit the concurrency of batches and error
		// if the limit would be exceeded.
		tickerCircuitBreaker := sync2.NewCircuitBreaker(maxConcurrentBatchLimit)
		// batchLimiter limits request concurrency to match the maximum supported by hardware.
		batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)

		batchNumber := new(atomic.Uint64)

		logger.Debug().Msg("running ticker loop")

	tickerLoop:
		for range time.NewTicker(time.Second).C {
			// Abort this tick's batch if the suite context was cancelled.
			select {
			case <-s.onChainStateChangeStartCh:
				clearLine(s)
				logger.Debug().Msg("pausing ticker loop")
				batchLimiter.Close()
				tickerCircuitBreaker.Close()
				break tickerLoop
			default:
			}

			relaysPerSec := s.relaysPerSec.Load()

			//clearLine(s)
			//logger.Debug().Msg("new tick")

			clearLine(s)
			logger.Debug().Msg("starting new batch")

			// Each batch should not block on any prior batch but if batches accumulate, error.
			startBatchFn, batchDoneCh := s.goStartRelayBatchFn(batchLimiter, batchNumber, relaysPerSec)
			ok := tickerCircuitBreaker.Go(s.ctx, startBatchFn)

			// If batches start to accumulate, they will likely never recover.
			require.Truef(s, ok, "batch limit exceeded: %d, reduce request runtime or increase request concurrency", maxConcurrentBatchLimit)

			<-batchDoneCh
			logger.Debug().Msg("batch done")

			// Cancel the suite context one batch after max relays per second is reached.
			if relaysPerSec == maxRelaysPerSec {
				s.cancelCtx()
			}
		}

		// Wait for the state change to complete before starting the next batch.
		<-s.onChainStateChangeCompleteCh
	}
}

// goStartRelayBatchFn starts a relay batch at a rate determined by the number of
// relays per second. It is intended to be run in a goroutine.
func (s *relaysSuite) goStartRelayBatchFn(
	batchLimiter *sync2.Limiter,
	batchNumber *atomic.Uint64,
	relaysPerSec int64,
) (start func(), doneCh <-chan struct{}) {
	batchDoneCh := make(chan struct{})

	return func() {
		batchWaitGroup := sync.WaitGroup{}
		remainingRelays := s.totalExpectedRequests - s.relaysComplete.Load()
		// Ensure the number of relays sent in this batch does not exceed the maximum.
		// I.e. this is the last batch.
		if remainingRelays < uint64(relaysPerSec) {
			relaysPerSec = int64(remainingRelays)
		}
		batchWaitGroup.Add(int(relaysPerSec))

		for i := int64(0); i < relaysPerSec; i++ {
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

		// TODO_IN_THIS_COMMIT: comment explaining why inc. batch number first.
		nextBatchNumber := batchNumber.Add(1)
		batchWaitGroup.Wait()
		close(batchDoneCh)

		clearLine(s)
		logger.Info().Msgf(
			"batch %d complete (%d/%d)",
			nextBatchNumber-1,
			relaysPerSec,
			relaysPerSec,
		)
		printProgressLine(s, progressBarWidth, s.relaysComplete.Load(), s.totalExpectedRequests)

	}, batchDoneCh
}

// goIncRelaysPerSec increments the number of relays per second to send every
// numBlocksInc blocks. It also ensures the number of relays per second to send
// does not exceed the maximum. It is intended to be run in a goroutine.
func goIncRelaysPerSec(
	s *relaysSuite,
	relaysPerSecondInc,
	numBlocksInc,
	maxRelaysPerSecond int64,
) {
	blocksCh := s.blocksReplayObs.Subscribe(s.ctx).Ch()

	// Synchronize initial increment counter with goStartRelayBatchTimer (next block height).
	for block := range blocksCh {
		if block.Height() <= s.startingBlockHeight {
			logger.Debug().Msg("skipping block in goIncRelaysPerSec")
			continue
		}

		clearLine(s)
		logger.Debug().Msgf("block height: %d", block.Height())

		// Every numBlocksInc, increment the number of relays to send per second.
		if block.Height()%numBlocksInc == 0 {
			// Ensure the number of relays to send per second does not exceed the maximum.
			nextRelaysPerSec := s.relaysPerSec.Load() + relaysPerSecondInc
			if nextRelaysPerSec > maxRelaysPerSecond {
				nextRelaysPerSec = maxRelaysPerSecond
			}

			// Update the number of relays per second to send
			s.relaysPerSec.Store(nextRelaysPerSec)
		}

	}
}

// goMonitorProgress monitors the progress of the relay requests by printing
// a progress bar to the console. It is intended to be run in a goroutine.
func (s *relaysSuite) goMonitorProgress() {
	s.Helper()

	for range time.NewTicker(time.Second / 10).C {
		// Abort monitoring if the context was cancelled.
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		relaysComplete := s.relaysComplete.Load()
		//totalExpectedRequests := uint64(s.relaysPerSec.Load()) * batchNumber.Load()

		printProgressLine(s, progressBarWidth, relaysComplete, s.totalExpectedRequests)
	}
}

// clearLine clears the current line in the console.
func clearLine(t gocuke.TestingT) {
	t.Helper()

	fmt.Printf("\r%s", strings.Repeat(" ", getTermWidth(t)))
	fmt.Print("\r")
}

// printProgressLine prints a progress bar to the console.
func printProgressLine(t gocuke.TestingT, barWidth, completeCount, totalCount uint64) {
	t.Helper()

	var completeChars, pendingChars uint64

	if totalCount != 0 {
		completeChars = barWidth * completeCount / totalCount
		pendingChars = barWidth - completeChars
	}

	if pendingChars+completeChars > barWidth {
		clearLine(t)
		logger.Warn().Msg("progress bar overflowed")
	}

	// Print the progress bar
	fmt.Printf(
		"\r[%s%s] (%d/%d)",
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
func computeTotalRequests(initialRelaysPerSec, relaysPerSecInc, numBlocksInc, maxRelaysPerSec int64) uint64 {
	var totalRequests uint64
	for rps := initialRelaysPerSec; rps <= maxRelaysPerSec; rps += relaysPerSecInc {
		totalRequests += uint64(rps * numBlocksInc * blocksPerSecond)
	}
	return totalRequests
}
