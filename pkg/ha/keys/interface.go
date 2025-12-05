package keys

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// KeyManager provides dynamic management of supplier signing keys.
// It supports hot-reload of keys without service restart.
type KeyManager interface {
	// GetSigner returns the private key for signing for a given operator address.
	// Returns error if no key is found for the address.
	GetSigner(operatorAddr string) (cryptotypes.PrivKey, error)

	// ListSuppliers returns all operator addresses that have signing keys.
	ListSuppliers() []string

	// HasKey returns true if a key exists for the given operator address.
	HasKey(operatorAddr string) bool

	// AddKey dynamically adds a new supplier key.
	// If a key already exists for this address, it is replaced.
	AddKey(operatorAddr string, key cryptotypes.PrivKey) error

	// RemoveKey removes a supplier key.
	// Returns error if the key doesn't exist.
	RemoveKey(operatorAddr string) error

	// Reload reloads keys from all configured sources.
	// This is called automatically on file changes if hot-reload is enabled.
	Reload(ctx context.Context) error

	// OnKeyChange registers a callback that is called when keys change.
	// The callback receives the operator address and whether the key was added (true) or removed (false).
	OnKeyChange(callback KeyChangeCallback)

	// Start starts background processes (file watching, etc.)
	Start(ctx context.Context) error

	// Close gracefully shuts down the key manager.
	Close() error
}

// KeyChangeCallback is called when a key is added or removed.
type KeyChangeCallback func(operatorAddr string, added bool)

// KeyProvider is a source of keys for the KeyManager.
// Multiple providers can be combined (keyring + file).
type KeyProvider interface {
	// Name returns a human-readable name for this provider.
	Name() string

	// LoadKeys loads all keys from this provider.
	// Returns a map of operator address -> private key.
	LoadKeys(ctx context.Context) (map[string]cryptotypes.PrivKey, error)

	// SupportsHotReload returns true if this provider supports hot-reload.
	SupportsHotReload() bool

	// WatchForChanges returns a channel that signals when keys may have changed.
	// Only called if SupportsHotReload returns true.
	WatchForChanges(ctx context.Context) <-chan struct{}

	// Close gracefully shuts down the provider.
	Close() error
}

// KeyManagerConfig contains configuration for the KeyManager.
type KeyManagerConfig struct {
	// KeyringBackend is the Cosmos keyring backend type.
	// Options: "file", "os", "test"
	KeyringBackend string

	// KeyringDir is the directory containing the keyring.
	// Default: ~/.pocket
	KeyringDir string

	// AdditionalKeysDir is an optional directory containing additional key files.
	// Key files are YAML/JSON with operator address and hex-encoded private key.
	AdditionalKeysDir string

	// HotReloadEnabled enables automatic key reload on file changes.
	HotReloadEnabled bool

	// HotReloadInterval is how often to check for file changes (if not using fsnotify).
	HotReloadInterval int64 // seconds
}
