package auth

import (
	cosmoserrors "cosmossdk.io/errors"
)

var (
	codespace                          = "client/auth"
	ErrQueryAccountNotFound            = cosmoserrors.Register(codespace, 1, "account not found")
	ErrQueryUnableToDeserializeAccount = cosmoserrors.Register(codespace, 2, "unable to deserialize account")
	ErrQueryPubKeyNotFound             = cosmoserrors.Register(codespace, 4, "account pub key not found")
)
