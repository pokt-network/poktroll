package block

import errorsmod "cosmossdk.io/errors"

var (
	ErrUnmarshalBlockEvent = errorsmod.Register(codespace, 1, "failed to unmarshal committed block event")
	codespace              = "block_client"
)
