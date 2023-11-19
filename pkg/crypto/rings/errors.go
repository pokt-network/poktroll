package rings

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace          = "rings"
	ErrRingsWrongCurve = sdkerrors.Register(codespace, 3, "key is not a secp256k1 public key")
)
