package observable

import (
	skderrors "cosmossdk.io/errors"
)

var (
	codespace = "observable"

	ErrObserverClosed                   = skderrors.Register(codespace, 1, "observer is closed")
	ErrMergeObservableMultipleFailModes = skderrors.Register(codespace, 2, "cannot use multiple fail modes")
)
