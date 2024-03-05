package proxy

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                        = "relayer_proxy"
	ErrRelayerProxyUnsupportedRPCType                = sdkerrors.Register(codespace, 1, "unsupported relayer proxy rpc type")
	ErrRelayerProxyInvalidSession                    = sdkerrors.Register(codespace, 2, "invalid session in relayer request")
	ErrRelayerProxyInvalidSupplier                   = sdkerrors.Register(codespace, 3, "invalid relayer proxy supplier")
	ErrRelayerProxyUndefinedSigningKeyName           = sdkerrors.Register(codespace, 4, "undefined relayer proxy signing key name")
	ErrRelayerProxyUndefinedProxiedServicesEndpoints = sdkerrors.Register(codespace, 5, "undefined proxied services endpoints for relayer proxy")
	ErrRelayerProxyInvalidRelayRequest               = sdkerrors.Register(codespace, 6, "invalid relay request")
	ErrRelayerProxyInvalidRelayResponse              = sdkerrors.Register(codespace, 7, "invalid relay response")
	ErrRelayerProxyServiceEndpointNotHandled         = sdkerrors.Register(codespace, 8, "service endpoint not handled by relayer proxy")
	ErrRelayerProxyUnsupportedTransportType          = sdkerrors.Register(codespace, 9, "unsupported proxy transport type")
)
