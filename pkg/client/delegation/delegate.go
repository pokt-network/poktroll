package delegation

import (
	"encoding/json"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

var _ client.DelegateeChange = (*delegate)(nil)

// delegate wraps the EventDelegateeChange event emitted by the application
// module, for use in the observable
type delegate struct {
	Address string `json:"app_address"`
}

// AppAddress returns the application address of the delegatee change event
func (d delegate) AppAddress() string {
	return d.Address
}

// newDelegateeChangeEvent attempts to deserialise the given bytes into a
// delegate struct. If the delegate struct has an empty app address then an
// ErrUnmarshalDelegateeChange error is returned. Otherwise if deserialisation
// fails then the error is returned.
func newDelegateeChangeEvent(delegateeChangeEventBz []byte) (client.DelegateeChange, error) {
	delegateeChange := new(delegate)
	if err := json.Unmarshal(delegateeChangeEventBz, delegateeChange); err != nil {
		return nil, err
	}

	if delegateeChange.Address == "" {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("unable to unmarshal delegatee change: %s", string(delegateeChangeEventBz))
	}

	return delegateeChange, nil
}
