package delegation

// TODO_TECHDEBT(#280): Refactor to use merged observables and subscribe to
// MsgDelegateToGateway and MsgUndelegateFromGateway messages directly, instead
// of listening to all events and doing a verbose filter.

import (
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// redelegationEventType is the type of the EventRedelegation event emitted by
// both the MsgDelegateToGateway and MsgUndelegateFromGateway messages.
var redelegationEventType = cosmostypes.MsgTypeURL(&apptypes.EventRedelegation{})

// newRedelegationEventFactoryFn is a factory function that returns a
// function that attempts to deserialize the given bytes into a redelegation
// struct. If the delegate struct has an empty app address then an
// ErrUnmarshalRedelegation error is returned. Otherwise if deserialisation
// fails then the error is returned.
func newRedelegationEventFactoryFn() events.NewEventsFn[*apptypes.EventRedelegation] {
	return func(eventBz []byte) (*apptypes.EventRedelegation, error) {
		// Try to deserialize the provided bytes into an abci.TxResult.
		txResult, err := tx.UnmarshalTxResult(eventBz)
		if err != nil {
			return nil, err
		}

		// Iterate through the log entries to find EventRedelegation
		for _, event := range txResult.Result.Events {
			if strings.Trim(event.GetType(), "/") != strings.Trim(redelegationEventType, "/") {
				continue
			}

			typedEvent, err := cosmostypes.ParseTypedEvent(event)
			if err != nil {
				return nil, err
			}

			redelegationEvent, ok := typedEvent.(*apptypes.EventRedelegation)
			if !ok {
				return nil, events.ErrEventsUnmarshalEvent.Wrapf("unexpected event type: %T", typedEvent)
			}

			// TODO_MAINNET(@bryanchriswhite): Refactor DelegationClient and/or ReplayClient to support multiple events per tx.
			return redelegationEvent, nil
		}
		return nil, events.ErrEventsUnmarshalEvent.Wrap("no redelegation event found")
	}
}
