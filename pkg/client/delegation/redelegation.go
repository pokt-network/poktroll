package delegation

import (
	"encoding/json"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

var _ client.Redelegation = (*redelegation)(nil)

// redelegation wraps the EventRedelegation event emitted by the application
// module, for use in the observable
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
		redelegationEvent := new(redelegation)
		if err := json.Unmarshal(redelegationEventBz, redelegationEvent); err != nil {
			return nil, err
		}

		if redelegationEvent.AppAddress == "" || redelegationEvent.GatewayAddress == "" {
			return nil, events.ErrEventsUnmarshalEvent.
				Wrapf("with redelegation: %s", string(redelegationEventBz))
		}

		return redelegationEvent, nil
	}
}
