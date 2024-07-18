package session

import (
	cosmoserrors "cosmossdk.io/errors"
)

var (
	codespace               = "client/session"
	ErrQueryRetrieveSession = cosmoserrors.Register(codespace, 3, "error while trying to retrieve a session")
)
