package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tokenomics module sentinel errors
var (
	ErrAuthorityInvalidAddress = sdkerrors.Register(ModuleName, 1, "provided authority address is invalid")
)
