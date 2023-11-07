package appgateserver

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                        = "appclient"
	ErrInvalidRelayResponseSignature = sdkerrors.Register(codespace, 1, "invalid relay response signature")
	ErrNoRelayEndpoints              = sdkerrors.Register(codespace, 2, "no relay endpoints found")
	ErrInvalidRequestURL             = sdkerrors.Register(codespace, 3, "invalid request URL")
)
