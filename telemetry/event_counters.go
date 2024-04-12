package telemetry

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
)

const eventSuccessKey = "event_type"

// EventSuccessCounter increments a counter with the given data type and success status.
func EventSuccessCounter(eventType string, isSuccessful func() bool) {
	successResult := strconv.FormatBool(isSuccessful())

	telemetry.IncrCounterWithLabels(
		[]string{eventSuccessKey},
		1.0,
		[]metrics.Label{
			{Name: "type", Value: eventType},
			{Name: "is_successful", Value: successResult},
		},
	)
}
