package keyring

import (
	"cosmossdk.io/errors"
)

var (
	// ErrEmptySigningKeyName represents an error which indicates that the
	// provided signing key name is empty or unspecified.
	ErrEmptySigningKeyName = errors.Register(codespace, 1, "empty signing key name")

	// ErrNoSuchSigningKey represents an error signifying that the requested
	// signing key does not exist or could not be located.
	ErrNoSuchSigningKey = errors.Register(codespace, 2, "signing key does not exist")

	// ErrSigningKeyAddr is raised when there's a failure in retrieving the
	// associated address for the provided signing key.
	ErrSigningKeyAddr = errors.Register(codespace, 3, "failed to get address for signing key")

	codespace = "keyring"
)
