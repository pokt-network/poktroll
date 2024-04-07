package telemetry

import (
	"context"
	"strconv"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/hashicorp/go-metrics"
)

// InitAppMetrics initializes the block specific metrics for the application.
func InitAppMetrics(app *baseapp.BaseApp) {
	app.SetPrepareProposal(initPrepareProposalHandlerWithMetrics(app))
	app.SetStreamingManager(initStreamingManagerWithMetrics())
}

// initPrepareProposalHandlerWithMetrics initializes the prepare proposal handler
// with the app metrics.
// It gathers the block txs size to emit them as a gauge metric.
func initPrepareProposalHandlerWithMetrics(
	app *baseapp.BaseApp,
) sdk.PrepareProposalHandler {
	// Create a NoOpMempool for the application and get the default prepare proposal
	// handler as per NewBaseApp implementation.
	// See https://github.com/cosmos/cosmos-sdk/blob/v0.50.4/baseapp/baseapp.go#L221
	app.SetMempool(mempool.NoOpMempool{})
	abciProposalHandler := baseapp.NewDefaultProposalHandler(app.Mempool(), app)
	prepareProposalHandler := abciProposalHandler.PrepareProposalHandler()

	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		// Iterate over the transactions in the block and calculate the block txs size.
		// This does not have access to non serializable transaction but can be used
		// to calculate the size of the transactions handled in the block.
		blockTxsSize := 0
		for _, tx := range req.Txs {
			blockTxsSize += len(tx)
		}

		telemetry.SetGaugeWithLabels(
			[]string{"block_txs_size"},
			float32(blockTxsSize),
			[]metrics.Label{
				{Name: "block_height", Value: strconv.FormatInt(req.Height, 10)},
			},
		)
		// Forward the request to the prepare proposal handler.
		return prepareProposalHandler(ctx, req)
	}
}

func initStreamingManagerWithMetrics() storetypes.StreamingManager {
	return storetypes.StreamingManager{
		ABCIListeners: []storetypes.ABCIListener{
			metricsABCIListener{},
		},
	}
}

type metricsABCIListener struct{}

func (mal metricsABCIListener) ListenFinalizeBlock(
	ctx context.Context,
	req abci.RequestFinalizeBlock,
	res abci.ResponseFinalizeBlock,
) error {
	telemetry.SetGaugeWithLabels(
		[]string{"block_result_size"},
		float32(res.Size()),
		[]metrics.Label{
			{Name: "block_height", Value: strconv.FormatInt(req.Height, 10)},
		},
	)
	return nil
}

func (mal metricsABCIListener) ListenCommit(
	ctx context.Context,
	res abci.ResponseCommit,
	changeSet []*storetypes.StoreKVPair,
) error {
	return nil
}
