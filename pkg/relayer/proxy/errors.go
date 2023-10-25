package proxy

import errorsmod "cosmossdk.io/errors"

var (
	ErrUnsupportedRPCType = errorsmod.Register(codespace, 1, "unsupported rpc type")
	ErrInvalidSignature   = errorsmod.Register(codespace, 2, "invalid signature")
	ErrInvalidSession     = errorsmod.Register(codespace, 3, "invalid session")
	ErrInvalidSupplier    = errorsmod.Register(codespace, 4, "invalid supplier")
	codespace             = "relayer/proxy"
)
