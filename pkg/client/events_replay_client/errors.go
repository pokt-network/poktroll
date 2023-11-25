package eventsreplayclient

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                           = "events_replay_client"
	ErrEventsReplayClientUnmarshalEvent = sdkerrors.Register(codespace, 1, "failed to unmarshal event bytes")
)
