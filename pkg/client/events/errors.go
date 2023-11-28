package events

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace = "events"

	ErrDial                 = sdkerrors.Register(codespace, 1, "dialing for connection failed")
	ErrConnClosed           = sdkerrors.Register(codespace, 2, "connection closed")
	ErrSubscribe            = sdkerrors.Register(codespace, 3, "failed to subscribe to events")
	ErrEventsUnmarshalEvent = sdkerrors.Register(codespace, 4, "failed to unmarshal event bytes")
)
