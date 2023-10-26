package proxy

import sdkerrors "cosmossdk.io/errors"

var (
	codespace             = "relayer/proxy"
	ErrUnsupportedRPCType = sdkerrors.Register(codespace, 1, "unsupported rpc type")
	ErrInvalidSignature   = sdkerrors.Register(codespace, 2, "invalid signature")
	ErrInvalidSession     = sdkerrors.Register(codespace, 3, "invalid session")
	ErrInvalidSupplier    = sdkerrors.Register(codespace, 4, "invalid supplier")
)
