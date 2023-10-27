package session

import sdkerrors "cosmossdk.io/errors"

var (
	codespace               = "relayer/session"
	ErrSessionTreeClosed    = sdkerrors.Register(codespace, 1, "session tree already closed")
	ErrSessionTreeNotClosed = sdkerrors.Register(codespace, 2, "session tree not closed")
)
