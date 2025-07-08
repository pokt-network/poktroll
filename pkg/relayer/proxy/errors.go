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
	ErrRelayerProxyCalculateRelayCost        = sdkerrors.Register(codespace, 8, "failed to calculate relay cost")
	ErrRelayerProxySupplierNotReachable      = sdkerrors.Register(codespace, 9, "supplier(s) not reachable")
	ErrRelayerProxyTimeout                   = sdkerrors.Register(codespace, 10, "relayer proxy request timed out")
	ErrRelayerProxyMaxBodyExceeded           = sdkerrors.Register(codespace, 11, "max body size exceeded")
	ErrRelayerProxyResponseLimitExceeded     = sdkerrors.Register(codespace, 12, "response limit exceed")
	ErrRelayerProxyRequestLimitExceeded      = sdkerrors.Register(codespace, 13, "request limit exceed")
	ErrRelayerProxyUnmarshalingRelayRequest  = sdkerrors.Register(codespace, 14, "failed to unmarshal relay request")
)
