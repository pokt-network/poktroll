package telemetry

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hashicorp/go-metrics"
)

const app_msg_type = "msg_type"

// AppMsgCounter increments a counter with the given data type and success status.
func AppMsgCounter(ctx context.Context, msgType string, isSuccessful func() bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	success := "false"
	if isSuccessful() {
		success = "true"
	}

	telemetry.IncrCounterWithLabels(
		[]string{app_msg_type},
		1.0,
		[]metrics.Label{
			{Name: "type", Value: msgType},
			{Name: "block_height", Value: strconv.FormatInt(blockHeight, 10)},
			{Name: "is_successful", Value: success},
		},
	)
}
