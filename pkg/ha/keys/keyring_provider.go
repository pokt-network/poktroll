package keys

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ KeyProvider = (*KeyringProvider)(nil)

// KeyringProvider loads keys from a Cosmos SDK keyring.
type KeyringProvider struct {
	logger  polylog.Logger
	keyring keyring.Keyring
	appName string

	// Optional: list of specific key names to load.
	// If empty, loads all keys from keyring.
	keyNames []string
}

// KeyringProviderConfig contains configuration for the KeyringProvider.
type KeyringProviderConfig struct {
	// Backend is the keyring backend type: "file", "os", "test", "memory"
	Backend string

	// Dir is the directory containing the keyring (for "file" backend).
	Dir string

	// AppName is the application name for the keyring.
	AppName string

	// KeyNames is an optional list of specific key names to load.
	// If empty, loads all keys from the keyring.
	KeyNames []string
}

// getKeyringCodec returns a codec for keyring operations.
func getKeyringCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

// NewKeyringProvider creates a new provider that reads from Cosmos keyring.
func NewKeyringProvider(
	logger polylog.Logger,
	config KeyringProviderConfig,
) (*KeyringProvider, error) {
	if config.AppName == "" {
		config.AppName = "pocket"
	}

	cdc := getKeyringCodec()

	// Create keyring based on backend type
	var kr keyring.Keyring
	var err error

	switch config.Backend {
	case "memory":
		kr = keyring.NewInMemory(cdc)
	case "test":
		// Test backend stores to disk but doesn't require password
		kr, err = keyring.New(
			config.AppName,
			keyring.BackendTest,
			config.Dir,
			nil, // No stdin for non-interactive
			cdc,
		)
	case "file":
		kr, err = keyring.New(
			config.AppName,
			keyring.BackendFile,
			config.Dir,
			nil, // No stdin for non-interactive
			cdc,
		)
	case "os":
		kr, err = keyring.New(
			config.AppName,
			keyring.BackendOS,
			config.Dir,
			nil,
			cdc,
		)
	default:
		return nil, fmt.Errorf("unsupported keyring backend: %s", config.Backend)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return &KeyringProvider{
		logger:   logging.ForComponent(logger, logging.ComponentKeyRingProvider),
		keyring:  kr,
		appName:  config.AppName,
		keyNames: config.KeyNames,
	}, nil
}

// NewKeyringProviderWithKeyring creates a provider with an existing keyring.
func NewKeyringProviderWithKeyring(
	logger polylog.Logger,
	kr keyring.Keyring,
	keyNames []string,
) *KeyringProvider {
	return &KeyringProvider{
		logger:   logging.ForComponent(logger, logging.ComponentKeyRingProvider),
		keyring:  kr,
		keyNames: keyNames,
	}
}

// Name returns a human-readable name for this provider.
func (p *KeyringProvider) Name() string {
	return "keyring"
}

// LoadKeys loads all keys from the keyring.
func (p *KeyringProvider) LoadKeys(ctx context.Context) (map[string]cryptotypes.PrivKey, error) {
	keys := make(map[string]cryptotypes.PrivKey)

	// If specific key names are provided, load only those
	if len(p.keyNames) > 0 {
		for _, name := range p.keyNames {
			privKey, addr, err := p.loadKeyByName(name)
			if err != nil {
				p.logger.Warn().
					Err(err).
					Str("key_name", name).
					Msg("failed to load key from keyring")
				keyLoadErrors.WithLabelValues("keyring").Inc()
				continue
			}
			keys[addr] = privKey
			p.logger.Debug().
				Str("key_name", name).
				Str("operator", addr).
				Msg("loaded key from keyring")
		}
	} else {
		// Load all keys from keyring
		records, err := p.keyring.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list keyring keys: %w", err)
		}

		for _, record := range records {
			privKey, addr, err := p.loadKeyByName(record.Name)
			if err != nil {
				p.logger.Warn().
					Err(err).
					Str("key_name", record.Name).
					Msg("failed to load key from keyring")
				keyLoadErrors.WithLabelValues("keyring").Inc()
				continue
			}
			keys[addr] = privKey
			p.logger.Debug().
				Str("key_name", record.Name).
				Str("operator", addr).
				Msg("loaded key from keyring")
		}
	}

	p.logger.Info().
		Int("loaded", len(keys)).
		Msg("loaded keys from keyring")

	return keys, nil
}

// loadKeyByName loads a single key by name and returns the private key and address.
func (p *KeyringProvider) loadKeyByName(name string) (cryptotypes.PrivKey, string, error) {
	// Get the key record
	record, err := p.keyring.Key(name)
	if err != nil {
		return nil, "", fmt.Errorf("key not found: %w", err)
	}

	// Get the address
	addr, err := record.GetAddress()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get address: %w", err)
	}

	// Export the armored private key
	armoredPrivKey, err := p.keyring.ExportPrivKeyArmorByAddress(addr, "")
	if err != nil {
		return nil, "", fmt.Errorf("failed to export armored private key: %w", err)
	}

	// Unarmor the private key
	privKey, _, err := crypto.UnarmorDecryptPrivKey(armoredPrivKey, "")
	if err != nil {
		return nil, "", fmt.Errorf("failed to unarmor private key: %w", err)
	}

	// Ensure it's a secp256k1 key
	secpPrivKey, ok := privKey.(*secp256k1.PrivKey)
	if !ok {
		return nil, "", fmt.Errorf("key %s is not a secp256k1 key", name)
	}

	return secpPrivKey, addr.String(), nil
}

// SupportsHotReload returns false - keyring doesn't support hot-reload.
func (p *KeyringProvider) SupportsHotReload() bool {
	return false
}

// WatchForChanges returns nil - keyring doesn't support hot-reload.
func (p *KeyringProvider) WatchForChanges(ctx context.Context) <-chan struct{} {
	return nil
}

// Close gracefully shuts down the provider.
func (p *KeyringProvider) Close() error {
	// Nothing to close for keyring
	return nil
}
