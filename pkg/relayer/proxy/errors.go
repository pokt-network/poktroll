package proxy

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                            = "relayer/proxy"
	ErrUnsupportedRPCType                = sdkerrors.Register(codespace, 1, "unsupported rpc type")
	ErrInvalidRelayRequestSignature      = sdkerrors.Register(codespace, 2, "invalid relay request signature")
	ErrInvalidSession                    = sdkerrors.Register(codespace, 3, "invalid session")
	ErrInvalidSupplier                   = sdkerrors.Register(codespace, 4, "invalid supplier")
	ErrUndefinedSigningKeyName           = sdkerrors.Register(codespace, 5, "undefined signing key name")
	ErrUndefinedProxiedServicesEndpoints = sdkerrors.Register(codespace, 6, "undefined proxied services endpoints")
)
