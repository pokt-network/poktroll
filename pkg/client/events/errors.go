package events

import sdkerrors "cosmossdk.io/errors"

var (
	codespace = "events"

	ErrEventsDial           = sdkerrors.Register(codespace, 1, "dialing for connection failed")
	ErrEventsConnClosed     = sdkerrors.Register(codespace, 2, "connection closed")
	ErrEventsSubscribe      = sdkerrors.Register(codespace, 3, "failed to subscribe to events")
	ErrEventsUnmarshalEvent = sdkerrors.Register(codespace, 4, "failed to unmarshal event bytes")
	ErrEventsConsClosed     = sdkerrors.Register(codespace, 5, "eventsqueryclient connection closed")
)
