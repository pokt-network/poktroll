package delegation

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                        = "delegation_client"
	ErrUnmarshalDelegateeChangeEvent = sdkerrors.Register(codespace, 1, "failed to unmarshal delegatee change event")
)
