package pubkeyclient

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                  = "pubkeyclient"
	ErrPubKeyClientEmptyPubKey = sdkerrors.Register(codespace, 1, "empty public key")
)
