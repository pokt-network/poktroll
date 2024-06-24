package protocol

import sdkerrors "cosmossdk.io/errors"

var (
	ErrDifficulty = sdkerrors.New(codespace, 1, "difficulty error")
	codespace     = "relayer/protocol"
)
