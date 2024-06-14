package partials

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                           = "partial"
	ErrPartialInvalidPayload            = sdkerrors.Register(codespace, 1, "invalid partial payload")
	ErrPartialUnrecognizedRequestFormat = sdkerrors.Register(codespace, 2, "unrecognized request format in partial payload")
)
