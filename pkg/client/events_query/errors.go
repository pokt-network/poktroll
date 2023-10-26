package eventsquery

import errorsmod "cosmossdk.io/errors"

var (
	ErrDial       = errorsmod.Register(codespace, 1, "dialing for connection failed")
	ErrConnClosed = errorsmod.Register(codespace, 2, "connection closed")
	ErrSubscribe  = errorsmod.Register(codespace, 3, "failed to subscribe to events")

	codespace = "events_query_client"
)
