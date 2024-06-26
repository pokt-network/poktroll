package events

import (
	"strconv"
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
		if event.Type != protoType {
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
