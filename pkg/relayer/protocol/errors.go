package protocol

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrDifficulty = errorsmod.New(codespace, 1, "difficulty error")
	codespace     = "relayer/protocol"
)
