package block

import sdkerrors "cosmossdk.io/errors"

var (
	codespace = "block"

	ErrUnmarshalBlockEvent       = sdkerrors.Register(codespace, 1, "failed to unmarshal block event")
	ErrUnmarshalBlockHeaderEvent = sdkerrors.Register(codespace, 2, "failed to unmarshal block header event")
)
