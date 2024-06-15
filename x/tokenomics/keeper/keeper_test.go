package keeper_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// getEvent verifies that there is exactly one event of type protoType in
// the given events and returns it.
// If there are 0 or more than 1 events of the given type, it fails the test.
func getEvent[T *proto.Message](t *testing.T, events cosmostypes.Events, protoType string) T {
	t.Helper()

	var parsedEvent proto.Message
	numExpectedEvents := 0
	for _, event := range events {
		switch event.Type {
		case protoType:
			var err error
			parsedEvent, err = cosmostypes.ParseTypedEvent(abci.Event(event))
			require.NoError(t, err)
			numExpectedEvents++
		default:
			continue
		}
	}

	if numExpectedEvents == 1 {
		castedEvent, ok := parsedEvent.(T)
		require.True(t, ok, "unexpected event type")
		return castedEvent
	}

	require.NotEqual(t, 1, numExpectedEvents, "Expected exactly one claim event")

	return nil
}
