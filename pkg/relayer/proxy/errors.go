package proxy

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                  = "relayer/proxy"
	ErrUnsupportedRPCType      = sdkerrors.Register(codespace, 1, "unsupported rpc type")
	ErrInvalidRelayRequest     = sdkerrors.Register(codespace, 2, "invalid relay request")
	ErrInvalidRequestSignature = sdkerrors.Register(codespace, 3, "invalid relay request signature")
	ErrInvalidRelayResponse    = sdkerrors.Register(codespace, 4, "invalid relay response")
)
