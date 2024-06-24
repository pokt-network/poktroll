package observable

import sdkerrors "cosmossdk.io/errors"

var (
	ErrObserverClosed = sdkerrors.Register(codespace, 1, "observer is closed")
	codespace         = "observable"
)
