package rings

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                                 = "rings"
	ErrRingsNotSecp256k1Curve                 = sdkerrors.Register(codespace, 1, "key is not a secp256k1 public key")
	ErrRingClientEmptyRelayRequestSignature   = sdkerrors.Register(codespace, 2, "empty relay request signature")
	ErrRingClientInvalidRelayRequest          = sdkerrors.Register(codespace, 3, "invalid relay request")
	ErrRingClientInvalidRelayRequestSignature = sdkerrors.Register(codespace, 4, "invalid relay request signature")
)
