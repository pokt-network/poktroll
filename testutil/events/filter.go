package events

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

// FilterEvents filters through all of the provided events and returns the ones of
// the matching type, after being casted to the right type.
func FilterEvents[T proto.Message](
	t *testing.T,
	allEvents cosmostypes.Events,
	protoType string,
) (parsedEvents []T) {
	t.Helper()

	for _, event := range allEvents {
		if event.Type != protoType {
			continue
		}
		parsedEvent, err := cosmostypes.ParseTypedEvent(abci.Event(event))
		require.NoError(t, err)
		require.NotNil(t, parsedEvent)

		castedEvent, ok := parsedEvent.(T)
		require.True(t, ok)

		parsedEvents = append(parsedEvents, castedEvent)
	}

	return parsedEvents
}
