package proxy

import errorsmod "cosmossdk.io/errors"

var (
	ErrUnsupportedRPCType = errorsmod.Register(codespace, 1, "unsupported rpc type")
	codespace             = "relayer/proxy"
)
