package session

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                              = "relayer_session"
	ErrSessionTreeClosed                   = sdkerrors.Register(codespace, 1, "session tree already closed")
	ErrSessionTreeNotClosed                = sdkerrors.Register(codespace, 2, "session tree not closed")
	ErrSessionTreeStorePathExists          = sdkerrors.Register(codespace, 3, "session tree store path already exists")
	ErrSessionTreeProofPathMismatch        = sdkerrors.Register(codespace, 4, "session tree proof path mismatch")
	ErrSessionTreeUndefinedStoresDirectory = sdkerrors.Register(codespace, 5, "session tree key-value store directory undefined for where they will be saved on disk")
	ErrSessionTreeAlreadyMarkedAsClaimed   = sdkerrors.Register(codespace, 6, "session tree already marked as claimed")
	ErrSupplierClientNotFound              = sdkerrors.Register(codespace, 7, "supplier client not found")
)
