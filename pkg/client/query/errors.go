package query

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                          = "query"
	ErrQueryAccountNotFound            = sdkerrors.Register(codespace, 1, "account not found")
	ErrQueryUnableToDeserialiseAccount = sdkerrors.Register(codespace, 2, "unable to deserialise account")
)
