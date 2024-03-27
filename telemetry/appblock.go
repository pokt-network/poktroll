package telemetry

import (
	"runtime"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/hashicorp/go-metrics"
)

type BlockTelemetry struct {
	startTime            time.Time
	initialBlockMemAlloc uint64
	blockTxsSize         int
}

// InitBlockMetrics initializes the block metrics for the application
// by initializing a BlockTelemetry struct and setting the prepare proposal,
// begin blocker, and end blocker handlers with the metrics.
func InitBlockMetrics(app *baseapp.BaseApp) {
	blockMetrics := &BlockTelemetry{
		startTime:            time.Now(),
		initialBlockMemAlloc: 0,
		blockTxsSize:         0,
	}

	app.SetPrepareProposal(initPrepareProposalHandlerWithMetrics(app, blockMetrics))
	app.SetPrecommiter(initPreCommitterWithMetrics(blockMetrics))
}

// initPrepareProposalHandlerWithMetrics initializes the prepare proposal handler
// with the block metrics.
// It gathers the block txs size to emit them as a gauge metric.
// It follows the NewBaseApp implementation of the PrepareProposalHandler by
// setting the mempool to NoOpMempool and returning the prepare proposal handler.
// that is then wrapped in a closure to gather the block txs size before handing
// it off to the prepare proposal handler.
func initPrepareProposalHandlerWithMetrics(
	app *baseapp.BaseApp,
	blockMetrics *BlockTelemetry,
) sdk.PrepareProposalHandler {
	// Create a NoOpMempool for the application and get the default prepare proposal
	// handler as per NewBaseApp implementation.
	// See https://github.com/cosmos/cosmos-sdk/blob/v0.50.4/baseapp/baseapp.go#L221
	app.SetMempool(mempool.NoOpMempool{})
	abciProposalHandler := baseapp.NewDefaultProposalHandler(app.Mempool(), app)
	prepareProposalHandler := abciProposalHandler.PrepareProposalHandler()

	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		blockMetrics.startTime = time.Now()
		m := new(runtime.MemStats)
		runtime.ReadMemStats(m)
		blockMetrics.initialBlockMemAlloc = m.Alloc

		// Iterate over the transactions in the block and calculate the block txs size.
		// This does not have access to non serializable transaction but can be used
		// to calculate the size of the transactions handled in the block.
		blockMetrics.blockTxsSize = 0
		for _, tx := range req.Txs {
			blockMetrics.blockTxsSize += len(tx)
		}

		telemetry.SetGaugeWithLabels(
			[]string{"block_txs_size"},
			float32(blockMetrics.blockTxsSize),
			[]metrics.Label{
				{Name: "block_height", Value: strconv.FormatInt(ctx.BlockHeight(), 10)},
			},
		)
		// Forward the request to the prepare proposal handler.
		return prepareProposalHandler(ctx, req)
	}
}

// initPreCommitterWithMetrics initializes the precommit handler with the block metrics.
// It calculates the block time and block memory difference and emits them as gauge metrics.
func initPreCommitterWithMetrics(blockMetrics *BlockTelemetry) sdk.Precommiter {
	return func(ctx sdk.Context) {
		blockHeight := strconv.FormatInt(ctx.BlockHeight(), 10)
		telemetry.SetGaugeWithLabels(
			[]string{"block_time"},
			float32(time.Since(blockMetrics.startTime).Seconds()),
			[]metrics.Label{
				{Name: "block_height", Value: blockHeight},
			},
		)

		m := new(runtime.MemStats)
		runtime.ReadMemStats(m)
		memDiff := m.Alloc - blockMetrics.initialBlockMemAlloc

		telemetry.SetGaugeWithLabels(
			[]string{"block_memory"},
			float32(memDiff),
			[]metrics.Label{
				{Name: "block_height", Value: blockHeight},
			},
		)
	}
}
