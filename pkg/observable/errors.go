package observable

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrObserverClosed = errorsmod.Register(codespace, 1, "observer is closed")
	codespace         = "observable"
)
