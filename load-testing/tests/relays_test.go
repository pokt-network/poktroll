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

	gatewayCount     atomic.Int64
	applicationCount atomic.Int64
	supplierCount    atomic.Int64
	relaysPerSecond  atomic.Int64
	nextRelaysPerSec chan int64

	totalExpectedRequests uint64
}

func TestLoadRelays(t *testing.T) {
	gocuke.NewRunner(t, &relaysSuite{}).Path(filepath.Join(".", "relays.feature")).Run()
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
}

func (s *relaysSuite) TheFollowingInitialActorsAreStaked(table gocuke.DataTable) {
	// Stake initial gateway(s)

	// Stake initial application(s)

	// Stake initial supplier(s)

}

func (s *relaysSuite) MoreActorsAreStakedAsFollows(table gocuke.DataTable) {
	blocksReplayObs := s.blockClient.CommittedBlocksSequence(s.ctx)

	gatewayIncrement := table.Cell(1, 1)
	gatewayBlocksPerIncrement := table.Cell(1, 2).Int64()
	maxGateways := table.Cell(1, 3)

	applicationIncrement := table.Cell(2, 1)
	applicationBlocksPerIncrement := table.Cell(2, 2).Int64()
	maxApplications := table.Cell(2, 3)

	supplierIncrement := table.Cell(3, 1)
	supplierBlocksPerIncrement := table.Cell(3, 2).Int64()
	maxSuppliers := table.Cell(3, 3)

	blocksCh := blocksReplayObs.Subscribe(s.ctx).Ch()
	go func() {
		for block := range blocksCh {
			if block.Height()%gatewayBlocksPerIncrement == 0 {
				// Concurrently stake gateways every increment blocks.
				gatewayCount := s.gatewayCount.Load()
				if s.gatewayCount.Load() < maxGateways.Int64() {
					gatewaysToStake := maxGateways.Int64() - gatewayCount
					if gatewaysToStake > gatewayIncrement.Int64() {
						gatewaysToStake = gatewayIncrement.Int64()
					}

					// Stake new gateways...
					// TODO: synchronize staking to start & complete in-between batches.
				}
			}

			if block.Height()%applicationBlocksPerIncrement == 0 {
				// Concurrently stake applications every increment blocks.
				applicationCount := s.applicationCount.Load()
				if s.applicationCount.Load() < maxApplications.Int64() {
					applicationsToStake := maxApplications.Int64() - applicationCount
					if applicationsToStake > applicationIncrement.Int64() {
						applicationsToStake = applicationIncrement.Int64()
					}

					// Stake new applications...
					// Re-delegate all applications...
					// TODO: strategy for distributing app delegations across more than 7 gateways.
					// TODO: synchronize staking to start & complete in-between batches.
				}
			}

			if block.Height()%supplierBlocksPerIncrement == 0 {
				// Concurrently stake suppliers every increment blocks.
				supplierCount := s.supplierCount.Load()
				if supplierCount < maxSuppliers.Int64() {
					suppliersToStake := maxSuppliers.Int64() - supplierCount
					if suppliersToStake > supplierIncrement.Int64() {
						suppliersToStake = supplierIncrement.Int64()
					}

					// Stake new suppliers...
					// TODO: synchronize staking to start & complete in-between batches.
				}
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
	go goMonitorProgress(s)

	// Concurrently send relay batches, fail if the limit is exceeded.
	go goStartRelayBatchTicker(s, maxConcurrentBatchLimit, maxRelaysPerSec)

	// Concurrently increment number of relays to send per second...
	go goIncRelaysPerSec(s, relaysPerSecondInc, numBlocksInc, maxRelaysPerSec)

	<-s.ctx.Done()
}

func goStartRelayBatchTicker(s *relaysSuite, maxConcurrentBatchLimit uint, maxRelaysPerSec int64) {
	// tickerCircuitBreaker is used to limit the concurrency of batches and error
	// if the limit would be exceeded.
	tickerCircuitBreaker := sync2.NewCircuitBreaker(maxConcurrentBatchLimit)
	// batchLimiter limits request concurrency to match the maximum supported by hardware.
	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)

	// Synchronize initial batch start with goIncRelaysPerSec (next block height)..
	blocksSubCtx, cancelBlocksSub := context.WithCancel(s.ctx)
	<-s.blocksReplayObs.Subscribe(blocksSubCtx).Ch()
	defer cancelBlocksSub()

	batchNumber := new(atomic.Uint64)

	for range time.NewTicker(time.Second).C {
		relaysPerSec := s.relaysPerSecond.Load()

		// TODO_IN_THIS_COMMIT: comment explaining why
		//select {
		//case relaysPerSec = <-s.nextRelaysPerSec:
		//default:
		//}

		clearLine(s)
		logger.Debug().Msg("new tick")

		// Abort this tick's batch if the suite context was cancelled.
		select {
		case <-s.ctx.Done():
			clearLine(s)
			logger.Debug().Msg("context done; closing limiters")
			batchLimiter.Close()
			tickerCircuitBreaker.Close()
			return
		default:
		}

		clearLine(s)
		logger.Debug().Msg("starting new batch")

		// Each batch should not block on any prior batch but if batches accumulate, error.
		startBatchFn, batchDoneCh := goStartRelayBatchFn(s, batchLimiter, batchNumber, relaysPerSec)
		ok := tickerCircuitBreaker.Go(s.ctx, startBatchFn)

		// If batches start to accumulate, they will likely never recover.
		require.Truef(s, ok, "batch limit exceeded: %d, reduce request runtime or increase request concurrency", maxConcurrentBatchLimit)

		// TODO: cancel one batch after max relays per second is reached...
		if relaysPerSec == maxRelaysPerSec {
			<-batchDoneCh
			s.cancelCtx()
		}
	}
}

func goStartRelayBatchFn(
	s *relaysSuite,
	batchLimiter *sync2.Limiter,
	batchNumber *atomic.Uint64,
	relaysPerSec int64,
) (start func(), done <-chan struct{}) {
	batchDoneCh := make(chan struct{})

	return func() {

		var (
			//batchNumber     = new(atomic.Uint64)
			//relaysPerSec = s.relaysPerSec.Load()
			batchWaitGroup = sync.WaitGroup{}
		)
		// TODO: RESUME HERE!!!
		// TODO: RESUME HERE!!!
		// TODO: RESUME HERE!!!
		// TODO_IN_THIS_COMMIT: calculate
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
				//numRelays := s.relaysPerSec.Load()
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

func goIncRelaysPerSec(
	s *relaysSuite,
	relaysPerSecondInc,
	numBlocksInc,
	maxRelaysPerSecond int64,
) {
	blocksCh := s.blocksReplayObs.Subscribe(s.ctx).Ch()

	// Synchronize initial increment counter with goStartRelayBatchTimer (next block height).
	<-blocksCh
	for block := range blocksCh {
		clearLine(s)
		logger.Debug().Msgf("block height: %d", block.Height())

		// Every numBlocksInc, increment the number of relays to send per second.
		if block.Height()%numBlocksInc == 0 {
			// Ensure the number of relays to send per second does not exceed the maximum.
			nextRelaysPerSec := s.relaysPerSecond.Load() + relaysPerSecondInc
			if nextRelaysPerSec > maxRelaysPerSecond {
				nextRelaysPerSec = maxRelaysPerSecond
			}

			// Set the number of relays to send per second...
			// TODO_IN_THIS_COMMIT: comment explaining why send on nextRelaysPerSec before storing
			//s.nextRelaysPerSec <- nextRelaysPerSec
			s.relaysPerSecond.Store(nextRelaysPerSec)
		}

	}
}

func goMonitorProgress(s *relaysSuite) {
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

func clearLine(t gocuke.TestingT) {
	t.Helper()

	fmt.Printf("\r%s", strings.Repeat(" ", getTermWidth(t)))
	fmt.Print("\r")
}

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

func getTermWidth(t gocuke.TestingT) int {
	t.Helper()

	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = defaultClearLineWidth
	}

	return width
}

func computeTotalRequests(initialRelaysPerSec, relaysPerSecInc, numBlocksInc, maxRelaysPerSec int64) uint64 {
	var totalRequests uint64
	for rps := initialRelaysPerSec; rps <= maxRelaysPerSec; rps += relaysPerSecInc {
		totalRequests += uint64(rps * numBlocksInc * blocksPerSecond)
	}
	return totalRequests
}
