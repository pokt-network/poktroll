package mappedclient

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                     = "mapped_client"
	ErrMappedClientUnmarshalEvent = sdkerrors.Register(codespace, 1, "failed to unmarshal event bytes")
)
