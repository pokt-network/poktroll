package query

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                          = "query"
	ErrQueryAccountNotFound            = sdkerrors.Register(codespace, 1, "account not found")
	ErrQueryUnableToDeserializeAccount = sdkerrors.Register(codespace, 2, "unable to deserialize account")
	ErrQueryRetrieveSession            = sdkerrors.Register(codespace, 3, "error while trying to retrieve a session")
	ErrQueryPubKeyNotFound             = sdkerrors.Register(codespace, 4, "account pub key not found")
	ErrQuerySessionParams              = sdkerrors.Register(codespace, 5, "unable to query session params")
)
