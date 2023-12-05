package delegation

import (
	"context"
	"encoding/json"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

var _ client.Redelegation = (*redelegation)(nil)

// redelegation wraps the EventRedelegation event emitted by the application
// module, for use in the observable
type redelegation struct {
	Address string `json:"app_address"`
}

// AppAddress returns the application address of the redelegation event
func (d redelegation) AppAddress() string {
	return d.Address
}

// newRedelegationEventFactoryFn is a factory function that returns a
// function that attempts to deserialise the given bytes into a redelegation
// struct. If the delegate struct has an empty app address then an
// ErrUnmarshalRedelegation error is returned. Otherwise if deserialisation
// fails then the error is returned.
func newRedelegationEventFactoryFn(ctx context.Context) events.NewEventsFn[client.Redelegation] {
	return func(redelegationEventBz []byte) (client.Redelegation, error) {
		redelegationEvent := new(redelegation)
		if err := json.Unmarshal(redelegationEventBz, redelegationEvent); err != nil {
			return nil, err
		}

		if redelegationEvent.Address == "" {
			return nil, events.ErrEventsUnmarshalEvent.
				Wrapf("with redelegation: %s", string(redelegationEventBz))
		}

		return redelegationEvent, nil
	}
}
