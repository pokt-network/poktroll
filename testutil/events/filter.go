package events

import (
	"strconv"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
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
		abciEvent := abci.Event(event)
		var parsedEvent proto.Message
		switch protoType {
		case "poktroll.tokenomics.EventRelayMiningDifficultyUpdated":
			parsedEvent = proto.Message(abciToRelayMiningDifficultyUpdatedEvent(t, abciEvent))
		default:
			var err error
			parsedEvent, err = cosmostypes.ParseTypedEvent(abciEvent)
			require.NoError(t, err)
		}
		require.NotNil(t, parsedEvent)

		castedEvent, ok := parsedEvent.(T)
		require.True(t, ok)

		parsedEvents = append(parsedEvents, castedEvent)
	}

	return parsedEvents
}

// TODO_TECHDEBT: The functions below were needed because `cosmostypes.ParseTypedEvent(*event)`
// throws the following:
// 	'json: error calling MarshalJSON for type json.RawMessage: invalid character 'E' looking for beginning of value'
//	 typedEvent, err := cosmostypes.ParseTypedEvent(*event)

// abciToRelayMiningDifficultyUpdatedEvent converts an abci.Event to a tokenomics.EventRelayMiningDifficultyUpdated
// NB: This was a ChatGPT generated function.
func abciToRelayMiningDifficultyUpdatedEvent(t *testing.T, event abci.Event) *tokenomicstypes.EventRelayMiningDifficultyUpdated {
	t.Helper()
	var relayMiningDifficultyUpdatedEvent tokenomicstypes.EventRelayMiningDifficultyUpdated
	for _, attr := range event.Attributes {
		unquotedValue, err := strconv.Unquote(string(attr.Value))
		// TODO_TECHDEBT: Unsure why/how this unrelated key ever becomes one of the attributes.
		if attr.Key != "mode" {
			require.NoError(t, err)
		}
		switch string(attr.Key) {
		case "service_id":
			relayMiningDifficultyUpdatedEvent.ServiceId = unquotedValue
		case "prev_target_hash":
			relayMiningDifficultyUpdatedEvent.PrevTargetHash = []byte(unquotedValue)
		case "new_target_hash":
			relayMiningDifficultyUpdatedEvent.NewTargetHash = []byte(unquotedValue)
		case "prev_num_relays_ema":
			prevNumRelaysEma, err := strconv.ParseUint(unquotedValue, 10, 64)
			require.NoError(t, err)
			relayMiningDifficultyUpdatedEvent.PrevNumRelaysEma = prevNumRelaysEma
		case "new_num_relays_ema":
			newNumRelaysEma, err := strconv.ParseUint(unquotedValue, 10, 64)
			require.NoError(t, err)
			relayMiningDifficultyUpdatedEvent.NewNumRelaysEma = newNumRelaysEma
		}
	}
	return &relayMiningDifficultyUpdatedEvent
}
