package rings

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                          = "rings"
	ErrRingsAccountNotFound            = sdkerrors.Register(codespace, 1, "account not found")
	ErrRingsUnableToDeserialiseAccount = sdkerrors.Register(codespace, 2, "unable to deserialise account")
	ErrRingsWrongCurve                 = sdkerrors.Register(codespace, 3, "key is not a secp256k1 public key")
)
