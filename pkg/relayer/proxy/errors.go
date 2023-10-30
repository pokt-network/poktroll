package proxy

import sdkerrors "cosmossdk.io/errors"

var (
	codespace             = "relayer/proxy"
	ErrUnsupportedRPCType = sdkerrors.Register(codespace, 1, "unsupported rpc type")
)
