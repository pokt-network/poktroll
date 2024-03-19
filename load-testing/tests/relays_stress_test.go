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

	relaysSent     atomic.Uint64
	relaysComplete atomic.Uint64

	startingBlockHeight int64
	gatewayCount        atomic.Int64
	appCount            atomic.Int64
	supplierCount       atomic.Int64
	relaysPerSecond     atomic.Int64
	nextRelaysPerSec    chan int64

	totalExpectedRequests uint64

	onChainStateChangeStartCh    chan struct{}
	onChainStateChangeCompleteCh chan struct{}
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
	s.onChainStateChangeStartCh = make(chan struct{}, 1)
	s.onChainStateChangeCompleteCh = make(chan struct{}, 1)
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
	gatewayInc := table.Cell(1, 1)
	gatewayBlocksPerInc := table.Cell(1, 2).Int64()
	maxGateways := table.Cell(1, 3)

	appInc := table.Cell(2, 1)
	applBlocksPerInc := table.Cell(2, 2).Int64()
	maxApps := table.Cell(2, 3)

	supplierInc := table.Cell(3, 1)
	supplierBlocksPerInc := table.Cell(3, 2).Int64()
	maxSuppliers := table.Cell(3, 3)

	blocksCh := s.blocksReplayObs.Subscribe(s.ctx).Ch()
	s.startingBlockHeight = s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
	logger.Debug().Int64("starting block height", s.startingBlockHeight).Send()

	go func() {
		for block := range blocksCh {
			var (
				currentHeight               = block.Height()
				changingOnChainState        atomic.Bool
				onChainStateChangeWaitGroup sync.WaitGroup
				gatewayCount                = s.gatewayCount.Load()
				gatewaysToStake             int64
				appCount                    = s.appCount.Load()
				appsToStake                 int64
				supplierCount               = s.supplierCount.Load()
				suppliersToStake            int64
			)

			// Skip the starting block height.
			if currentHeight <= s.startingBlockHeight {
				continue
			}

			// Compute the number of gateways to stake add to the wait group.
			// The wait group will be "done"d when staking is complete.
			if currentHeight%gatewayBlocksPerInc == 0 {
				// Concurrently stake gateways every increment blocks.
				if gatewayCount < maxGateways.Int64() {
					gatewaysToStake = maxGateways.Int64() - gatewayCount
					if gatewaysToStake > gatewayInc.Int64() {
						gatewaysToStake = gatewayInc.Int64()
					}

					onChainStateChangeWaitGroup.Add(1)
				}
			}

			// Compute the number of applicaitons to stake add to the wait group.
			// The wait group will be "done"d when staking is complete.
			if currentHeight%applBlocksPerInc == 0 {
				// Concurrently stake applications every increment blocks.
				if appCount < maxApps.Int64() {
					appsToStake = maxApps.Int64() - appCount
					if appsToStake > appInc.Int64() {
						appsToStake = appInc.Int64()
					}

					onChainStateChangeWaitGroup.Add(1)
				}
			}

			// Compute the number of suppliers to stake add to the wait group.
			// The wait group will be "done"d when staking is complete.
			if currentHeight%supplierBlocksPerInc == 0 {
				// Concurrently stake suppliers every increment blocks.
				if supplierCount < maxSuppliers.Int64() {
					suppliersToStake = maxSuppliers.Int64() - supplierCount
					if suppliersToStake > supplierInc.Int64() {
						suppliersToStake = supplierInc.Int64()
					}

					onChainStateChangeWaitGroup.Add(1)
				}
			}

			if gatewaysToStake > 0 {
				nextGatewayCount := gatewayCount + gatewaysToStake

				clearLine(s)
				logger.Info().Msgf(
					"incrementing gateways (staking %d->%d)",
					gatewayCount,
					nextGatewayCount,
				)

				// Concurrently stake all new gateways.
				for gwIdx := int64(0); gwIdx < gatewaysToStake; gwIdx++ {
					go func(nextGatewayCount int64) {

						// Stake new gateways...
						// TODO: synchronize staking to start & complete in-between batches.
						changingOnChainState.CompareAndSwap(false, true)
						time.Sleep(2000)

						// Update gateway count after staking is completed.
						s.gatewayCount.Store(nextGatewayCount)

						onChainStateChangeWaitGroup.Done()
					}(nextGatewayCount)
				}
			}

			if appsToStake > 0 {
				nextAppCount := appCount + appsToStake

				clearLine(s)
				logger.Info().Msgf(
					"incrementing applications (staking %d->%d)",
					appCount,
					nextAppCount,
				)

				// Concurrently stake and delegate all new applications.
				for appIdx := int64(0); appIdx < appsToStake; appIdx++ {
					go func(nextApplicationCount int64) {

						// Stake new applications...
						// Re-delegate all applications...
						// TODO: strategy for distributing app delegations across more than 7 gateways.
						// TODO: synchronize staking to start & complete in-between batches.
						changingOnChainState.CompareAndSwap(false, true)
						time.Sleep(2000)

						s.appCount.Store(nextApplicationCount)

						onChainStateChangeWaitGroup.Done()
					}(nextAppCount)
				}
			}

			if suppliersToStake > 0 {
				nextSupplierCount := supplierCount + suppliersToStake

				clearLine(s)
				logger.Info().Msgf(
					"incrementing suppliers (staking %d->%d)",
					supplierCount,
					nextSupplierCount,
				)

				for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
					go func(nextSupplierCount int64) {

						// Stake new suppliers...
						// TODO: synchronize staking to start & complete in-between batches.
						changingOnChainState.CompareAndSwap(false, true)
						time.Sleep(2000)

						s.supplierCount.Store(nextSupplierCount)

						onChainStateChangeWaitGroup.Done()
					}(nextSupplierCount)
				}
			}

			// TODO_IN_THIS_COMMIT: something better than waiting...
			time.Sleep(100 * time.Millisecond)

			if changingOnChainState.Load() {
				clearLine(s)
				logger.Debug().Msg("on-chain state is changing...")
				//s.onChainStateChangeStartCh <- struct{}{}

				time.Sleep(2000)
				//onChainStateChangeWaitGroup.Wait()
				//
				//s.onChainStateChangeCompleteCh <- struct{}{}
				clearLine(s)
				logger.Debug().Msg("on-chain state done changing")
			}
		}
	}()
}

func (s *relaysSuite) ALoadOfConcurrentRelayRequestsAreSentPerSecondAsFollows(table gocuke.DataTable) {
	// Set initial relays per second
	initialRelaysPerSecond := table.Cell(1, 0).Int64()
	s.relaysPerSecond.Store(initialRelaysPerSecond)

	relaysPerSecondInc := table.Cell(1, 1).Int64()
	numBlocksInc := table.Cell(1, 2).Int64()
	maxRelaysPerSec := table.Cell(1, 3).Int64()

	s.totalExpectedRequests = computeTotalRequests(initialRelaysPerSecond, relaysPerSecondInc, numBlocksInc, maxRelaysPerSec)

	// Concurrently monitor total relay progress.
	go s.goMonitorProgress()

	// Concurrently send relay batches.
	go s.goStartRelayBatchTicker(maxConcurrentBatchLimit, maxRelaysPerSec)

	// Concurrently increment number of relays per second to send.
	go goIncRelaysPerSec(s, relaysPerSecondInc, numBlocksInc, maxRelaysPerSec)

	<-s.ctx.Done()
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

			relaysPerSec := s.relaysPerSecond.Load()

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
			nextRelaysPerSec := s.relaysPerSecond.Load() + relaysPerSecondInc
			if nextRelaysPerSec > maxRelaysPerSecond {
				nextRelaysPerSec = maxRelaysPerSecond
			}

			// Update the number of relays per second to send
			s.relaysPerSecond.Store(nextRelaysPerSec)
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
		//totalExpectedRequests := uint64(s.relaysPerSecond.Load()) * batchNumber.Load()

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
