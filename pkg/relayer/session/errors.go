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
	ErrSessionSupplierClientNotFound       = sdkerrors.Register(codespace, 7, "supplier client not found")
	ErrSessionUpdatingTree                 = sdkerrors.Register(codespace, 8, "error updating session SMST")
	ErrSessionRelayMetaHasNoServiceID      = sdkerrors.Register(codespace, 9, "service ID not specified in relay metadata")
	ErrSessionRelayMetaHasInvalidServiceID = sdkerrors.Register(codespace, 10, "service specified in relay metadata not found")
	ErrSessionPersistRelay                 = sdkerrors.Register(codespace, 11, "error persisting relay bytes")
	ErrSessionTreeNoProof                  = sdkerrors.Register(codespace, 12, "no proof found for session tree")
	ErrSessionRelaysStorePathExists        = sdkerrors.Register(codespace, 13, "relays store directory already exists")
)
