package event

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace              = "event"
	ErrEventUnmarshalEvent = sdkerrors.Register(codespace, 1, "failed to unmarshal event bytes")
)
