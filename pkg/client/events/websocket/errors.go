package websocket

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace = "events_query_client_websocket_connection"

	ErrEventsWebsocketReceive = sdkerrors.Register(codespace, 1, "failed to receive event over websocket connection to pocket node")
)
