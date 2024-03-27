package telemetry

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hashicorp/go-metrics"
)

// StateDataCounter increments a counter with the given data type and success status.
func StateDataCounter(ctx context.Context, dataType string, isSuccessful func() bool) {
	success := "false"
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	if isSuccessful() {
		success = "true"
	}

	telemetry.IncrCounterWithLabels(
		[]string{"state_data"},
		1.0,
		[]metrics.Label{
			{Name: "type", Value: dataType},
			{Name: "block_height", Value: strconv.FormatInt(blockHeight, 10)},
			{Name: "is_successful", Value: success},
		},
	)
}
