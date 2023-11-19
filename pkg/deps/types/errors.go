package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                         = "deps"
	ErrDepsAccountNotFound            = sdkerrors.Register(codespace, 1, "account not found")
	ErrDepsUnableToDeserialiseAccount = sdkerrors.Register(codespace, 2, "unable to deserialise account")
)
