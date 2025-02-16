package proxy

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                                = "relayer_proxy"
	ErrRelayerProxyInvalidSession            = sdkerrors.Register(codespace, 1, "invalid session in relayer request")
	ErrRelayerServicesConfigsUndefined       = sdkerrors.Register(codespace, 2, "services configurations are undefined")
	ErrRelayerProxyServiceEndpointNotHandled = sdkerrors.Register(codespace, 3, "service endpoint not handled by relayer proxy")
	ErrRelayerProxyUnsupportedTransportType  = sdkerrors.Register(codespace, 4, "unsupported proxy transport type")
	ErrRelayerProxyInternalError             = sdkerrors.Register(codespace, 5, "internal error")
	ErrRelayerProxyUnknownSession            = sdkerrors.Register(codespace, 6, "relayer proxy encountered unknown session")
	ErrRelayerProxyRateLimited               = sdkerrors.Register(codespace, 7, "offchain rate limit hit by relayer proxy")
	ErrRelayerProxyUnclaimRelayPrice         = sdkerrors.Register(codespace, 8, "failed to unclaim relay price")
)
