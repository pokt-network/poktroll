package websocket

import errorsmod "cosmossdk.io/errors"

var (
	ErrReceive = errorsmod.Register(codespace, 4, "failed to receive event")
	codespace  = "events_query_client_websocket_connection"
)
