package partials

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                           = "partial"
	ErrPartialUnrecognisedRequestFormat = sdkerrors.Register(
		codespace,
		1,
		"unrecognised request format",
	)
)
