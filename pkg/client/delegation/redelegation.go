package delegation

// TODO_TECHDEBT(@h5law): This is disgusting get this piece of shit out the codebase

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ client.Redelegation = (*redelegation)(nil)

// Define the structure of a response from the application module query
type response struct {
	Result result `json:"result"`
}
type result struct {
	Log string `json:"log"`
}
type logEntry struct {
	Events []event `json:"events"`
}
type event struct {
	Type       string      `json:"type"`
	Attributes []attribute `json:"attributes"`
}
type attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// redelegation wraps the EventRedelegation event emitted by the application
// module, for use in the observable, it is one of the log entries embedded
// within the log field of the response struct from the app module's query.
type redelegation struct {
	AppAddress     string `json:"app_address"`
	GatewayAddress string `json:"gateway_address"`
}

// GetAppAddress returns the application address of the redelegation event
func (d redelegation) GetAppAddress() string {
	return d.AppAddress
}

// GetGatewayAddress returns the gateway address of the redelegation event
func (d redelegation) GetGatewayAddress() string {
	return d.GatewayAddress
}

// newRedelegationEventFactoryFn is a factory function that returns a
// function that attempts to deserialize the given bytes into a redelegation
// struct. If the delegate struct has an empty app address then an
// ErrUnmarshalRedelegation error is returned. Otherwise if deserialisation
// fails then the error is returned.
func newRedelegationEventFactoryFn() events.NewEventsFn[client.Redelegation] {
	return func(redelegationEventBz []byte) (client.Redelegation, error) {
		var response response
		err := json.Unmarshal(redelegationEventBz, &response)
		if err != nil {
			return nil, events.ErrEventsUnmarshalEvent.
				Wrapf("unable to unmarshal subscription response: %v", err)
		}
		if response.Result.Log == "" {
			return nil, events.ErrEventsUnmarshalEvent.Wrap("no log field in response")
		}

		// Unmarshal the log field
		var logEntries []logEntry
		err = json.Unmarshal([]byte(response.Result.Log), &logEntries)
		if err != nil {
			return nil, events.ErrEventsUnmarshalEvent.
				Wrapf("unable to unmarshal log field in response: %v", err)
		}

		logger := polylog.Ctx(context.Background())
		// Iterate through the log entries to find EventRedelegation
		for _, entry := range logEntries {
			for _, event := range entry.Events {
				if event.Type == "pocket.application.EventRedelegation" {
					var redelegationEvent redelegation
					for _, attr := range event.Attributes {
						switch attr.Key {
						case "app_address":
							appAddress, err := unescape(attr.Value)
							if err != nil {
								return nil, events.ErrEventsUnmarshalEvent.
									Wrapf("unable to unescape app address string: %v", err)
							}
							redelegationEvent.AppAddress = appAddress
						case "gateway_address":
							gatewayAddr, err := unescape(attr.Value)
							if err != nil {
								return nil, events.ErrEventsUnmarshalEvent.
									Wrapf("unable to unescape gateway address string: %v", err)
							}
							redelegationEvent.GatewayAddress = gatewayAddr
						}
					}
					// Handle the redelegation event
					if redelegationEvent.AppAddress == "" || redelegationEvent.GatewayAddress == "" {
						return nil, events.ErrEventsUnmarshalEvent.
							Wrapf("with redelegation: %s", string(redelegationEventBz))
					}
					logger.Debug().
						Str("app_address", redelegationEvent.GetAppAddress()).
						Str("gateway_address", redelegationEvent.GetGatewayAddress()).
						Msg("redelegation event received")
					return redelegationEvent, nil
				}
			}
		}
		return nil, events.ErrEventsUnmarshalEvent.Wrap("no redelegation event found in log")
	}
}

func unescape(str string) (string, error) {
	// Convert the doubly-escaped string into a standard Go string literal
	processedStr := strings.Replace(str, `\\`, `\`, -1)
	// Use strconv.Unquote to unescape the string
	return strconv.Unquote(processedStr)
}
