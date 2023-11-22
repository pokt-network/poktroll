package partials

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                           = "partial"
	ErrPartialInvalidPayload            = sdkerrors.Register(codespace, 1, "invalid payload")
	ErrPartialUnrecognisedRequestFormat = sdkerrors.Register(codespace, 2, "unrecognised request format")
)
