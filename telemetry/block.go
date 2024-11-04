package telemetry

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	blockTxsSizeBytesMetric    = "block_txs_size_bytes"
	blockResultSizeBytesMetric = "block_result_size_bytes"
)

var (
	_                storetypes.ABCIListener = (*metricsABCIListener)(nil)
	DefaultCounterFn                         = func() float32 { return 1 }
)

// InitBlockMetrics initializes the block-specific metrics for the application.
func InitBlockMetrics(app *baseapp.BaseApp) {
	app.SetPrepareProposal(initPrepareProposalHandlerWithMetrics(app))
	app.SetStreamingManager(initStreamingManagerWithMetrics())
}

// initPrepareProposalHandlerWithMetrics initializes the prepare proposal handler
// with the app metrics.
// It gathers the block txs size to emit them as a gauge metric.
func initPrepareProposalHandlerWithMetrics(app *baseapp.BaseApp) sdk.PrepareProposalHandler {
	// Create a new default proposal handler to get the prepare proposal handler.
	// Since we are setting a prepare proposal handler, NewBaseApp will not set the
	// default one, requiring us to manually create it along its dependencies.
	// See https://github.com/cosmos/cosmos-sdk/blob/v0.50.4/baseapp/baseapp.go#L214-L221
	abciProposalHandler := baseapp.NewDefaultProposalHandler(app.Mempool(), app)
	prepareProposalHandler := abciProposalHandler.PrepareProposalHandler()

	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		// Iterate over the transactions in the block and calculate the block txs size.
		// This does not have access to non-serializable (i.e. invalid) txs but can
		// be used to calculate the size of the transactions handled in the block.
		blockTxsSizeBytes := 0
		for _, tx := range req.Txs {
			blockTxsSizeBytes += len(tx)
		}

		telemetry.SetGauge(
			float32(blockTxsSizeBytes),
			blockTxsSizeBytesMetric,
		)

		// Forward the request to the prepare proposal handler.
		return prepareProposalHandler(ctx, req)
	}
}

// initStreamingManagerWithMetrics initializes the streaming manager that listens
// for finalize block events to capture ResponseFinalizeBlock size.
func initStreamingManagerWithMetrics() storetypes.StreamingManager {
	return storetypes.StreamingManager{
		ABCIListeners: []storetypes.ABCIListener{
			metricsABCIListener{},
		},
	}
}

// metricsABCIListener is an implementation of the StreamingManager that hooks
// into ListenFinalizeBlock to capture ResponseFinalizeBlock size.
type metricsABCIListener struct{}

// ListenFinalizeBlock captures the ResponseFinalizeBlock size and emits it as a
// gauge metric.
func (mal metricsABCIListener) ListenFinalizeBlock(
	ctx context.Context,
	req abci.RequestFinalizeBlock,
	res abci.ResponseFinalizeBlock,
) error {
	if !isTelemetyEnabled() {
		return nil
	}

	telemetry.SetGauge(
		float32(res.Size()),
		blockResultSizeBytesMetric,
	)

	return nil
}

// ListenCommit is a no-op implementation of the ABCIListener's ListenCommit
// method.
// It is needed to adhere to the ABCIListener interface requiring the
// ListenCommit to be implemented.
func (mal metricsABCIListener) ListenCommit(
	ctx context.Context,
	res abci.ResponseCommit,
	changeSet []*storetypes.StoreKVPair,
) error {
	if !isTelemetyEnabled() {
		return nil
	}

	return nil
}
