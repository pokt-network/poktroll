package keyring

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	// ErrEmptySigningKeyName represents an error which indicates that the
	// provided signing key name is empty or unspecified.
	ErrEmptySigningKeyName = sdkerrors.Register(codespace, 1, "empty signing key name")

	// ErrNoSuchSigningKey represents an error signifying that the requested
	// signing key does not exist or could not be located.
	ErrNoSuchSigningKey = sdkerrors.Register(codespace, 2, "signing key does not exist")

	// ErrSigningKeyAddr is raised when there's a failure in retrieving the
	// associated address for the provided signing key.
	ErrSigningKeyAddr = sdkerrors.Register(codespace, 3, "failed to get address for signing key")

	codespace = "keyring"
)
