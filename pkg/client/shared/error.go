package shared

import (
	cosmoserrors "cosmossdk.io/errors"
)

var (
	codespace             = "client/shared"
	ErrQuerySessionParams = cosmoserrors.Register(codespace, 5, "unable to query session params")
)
