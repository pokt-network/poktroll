package websockets

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                          = "websockets"
	ErrWebsocketsConnection            = sdkerrors.Register(codespace, 1, "websockets connection error")
	ErrWebsocketsBridge                = sdkerrors.Register(codespace, 2, "websockets bridge error")
	ErrWebsocketsGatewayMessage        = sdkerrors.Register(codespace, 3, "websockets gateway message error")
	ErrWebsocketsServiceBackendMessage = sdkerrors.Register(codespace, 4, "websockets service backend message error")
)
