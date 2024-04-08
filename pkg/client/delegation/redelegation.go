package delegation

// TODO_TECHDEBT(#280): Refactor to use merged observables and subscribe to
// MsgDelegateToGateway and MsgUndelegateFromGateway messages directly, instead
// of listening to all events and doing a verbose filter.

import (
	"strconv"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
)

// redelegationEventType is the type of the EventRedelegation event emitted by
// both the MsgDelegateToGateway and MsgUndelegateFromGateway messages.
const redelegationEventType = "pocket.application.EventRedelegation"

var _ client.Redelegation = (*redelegation)(nil)

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
	return func(eventBz []byte) (client.Redelegation, error) {
		// Try to deserialize the provided bytes into an abci.TxResult.
		txResult, err := tx.UnmarshalTxResult(eventBz)
		if err != nil {
			return nil, err
		}

		// Iterate through the log entries to find EventRedelegation
		for _, event := range txResult.Result.Events {
			if event.GetType() != redelegationEventType {
				continue
			}
			var redelegationEvent redelegation
			for _, attr := range event.Attributes {
				switch attr.Key {
				case "app_address":
					appAddr, err := unescape(attr.Value)
					if err != nil {
						return nil, events.ErrEventsUnmarshalEvent.Wrapf("cannot retrieve app address: %v", err)
					}
					redelegationEvent.AppAddress = appAddr
				case "gateway_address":
					gatewayAddr, err := unescape(attr.Value)
					if err != nil {
						return nil, events.ErrEventsUnmarshalEvent.Wrapf("cannot retrieve gateway address: %v", err)
					}
					redelegationEvent.GatewayAddress = gatewayAddr
				default:
					return nil, events.ErrEventsUnmarshalEvent.Wrapf("unknown attribute key: %s", attr.Key)
				}
			}
			// Handle the redelegation event
			if redelegationEvent.AppAddress == "" || redelegationEvent.GatewayAddress == "" {
				return nil, events.ErrEventsUnmarshalEvent.
					Wrapf("empty redelegation event: %s", string(eventBz))
			}
			return redelegationEvent, nil
		}
		return nil, events.ErrEventsUnmarshalEvent.Wrap("no redelegation event found")
	}
}

func unescape(s string) (string, error) {
	return strconv.Unquote(s)
}
