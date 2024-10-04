package events

import (
	"strconv"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

// FilterEvents filters allEvents, returning list of T type events whose protobuf  message type string matches protoType.
func FilterEvents[T proto.Message](
	t *testing.T,
	allEvents cosmostypes.Events,
	protoType string,
) (parsedEvents []T) {
	t.Helper()

	for _, event := range allEvents.ToABCIEvents() {
		if event.Type != strings.Trim(protoType, "/") {
			continue
		}
		QuoteEventMode(&event)
		parsedEvent, err := cosmostypes.ParseTypedEvent(event)
		require.NoError(t, err)
		require.NotNil(t, parsedEvent)

		castedEvent, ok := parsedEvent.(T)
		require.True(t, ok)

		parsedEvents = append(parsedEvents, castedEvent)
	}

	return parsedEvents
}

// QuoteEventMode quotes (i.e. URL escape) the value associated with the 'mode'
// key in the event. This is injected by the caller that emits the event and
// causes issues in calling 'ParseTypedEvent'.
func QuoteEventMode(event *abci.Event) {
	for i, attr := range event.Attributes {
		if attr.Key == "mode" {
			event.Attributes[i].Value = strconv.Quote(attr.Value)
			return
		}
	}
}

// NewMsgEventMatchFn returns a function that matches events whose type equals
// the given event (protobuf message) type URL.
func NewMsgEventMatchFn(matchMsgTypeURL string) func(*cosmostypes.Event) bool {
	return func(event *cosmostypes.Event) bool {
		if event.Type != "message" {
			return false
		}

		actionAttr, hasActionAttr := event.GetAttribute("action")
		if !hasActionAttr {
			return false
		}

		eventMsgTypeURL := strings.Trim(actionAttr.GetValue(), "\"")
		return strings.Trim(eventMsgTypeURL, "/") == strings.Trim(matchMsgTypeURL, "/")
	}
}

// NewEventTypeMatchFn returns a function that matches events whose type is "message"
// and whose "action" attribute matches the given message (protobuf message) type URL.
func NewEventTypeMatchFn(matchEventType string) func(*cosmostypes.Event) bool {
	return func(event *cosmostypes.Event) bool {
		return strings.Trim(event.Type, "/") == strings.Trim(matchEventType, "/")
	}
}
