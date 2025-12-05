package keys

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ KeyProvider = (*SupplierKeysFileProvider)(nil)

// SupplierKeysFile is the structure of the supplier.yaml file.
// It contains a simple list of hex-encoded private keys.
// The operator address is derived from each private key.
type SupplierKeysFile struct {
	// Keys is a list of hex-encoded secp256k1 private keys.
	// Can be prefixed with "0x" or not.
	Keys []string `yaml:"keys" json:"keys"`
}

// SupplierKeysFileProvider loads keys from a single supplier.yaml file
// containing a list of hex-encoded private keys.
// The operator address is derived from each key.
type SupplierKeysFileProvider struct {
	logger   polylog.Logger
	filePath string
	watcher  *fsnotify.Watcher
	changeCh chan struct{}

	mu     sync.Mutex
	closed bool
}

// NewSupplierKeysFileProvider creates a new provider that reads from supplier.yaml.
func NewSupplierKeysFileProvider(logger polylog.Logger, filePath string) (*SupplierKeysFileProvider, error) {
	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("supplier keys file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("failed to stat supplier keys file: %w", err)
	}

	// Create fsnotify watcher for the file
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(filePath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch supplier keys file: %w", err)
	}

	return &SupplierKeysFileProvider{
		logger:   logging.ForComponent(logger, logging.ComponentSupplierKeysFile),
		filePath: filePath,
		watcher:  watcher,
		changeCh: make(chan struct{}, 1),
	}, nil
}

// Name returns a human-readable name for this provider.
func (p *SupplierKeysFileProvider) Name() string {
	return "supplier_keys_file:" + p.filePath
}

// LoadKeys loads all keys from the supplier.yaml file.
// The operator address is derived from each private key.
func (p *SupplierKeysFileProvider) LoadKeys(ctx context.Context) (map[string]cryptotypes.PrivKey, error) {
	keys := make(map[string]cryptotypes.PrivKey)

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier keys file: %w", err)
	}

	var keysFile SupplierKeysFile
	if err := yaml.Unmarshal(data, &keysFile); err != nil {
		return nil, fmt.Errorf("failed to parse supplier keys file: %w", err)
	}

	for i, hexKey := range keysFile.Keys {
		privKey, operatorAddr, err := parseHexKeyWithAddress(hexKey)
		if err != nil {
			p.logger.Warn().
				Err(err).
				Int("index", i).
				Msg("failed to parse key from supplier.yaml")
			keyLoadErrors.WithLabelValues("supplier_keys_file").Inc()
			continue
		}

		keys[operatorAddr] = privKey
		p.logger.Debug().
			Int("index", i).
			Str("operator", operatorAddr).
			Msg("loaded key from supplier.yaml")
	}

	p.logger.Info().
		Int("total_in_file", len(keysFile.Keys)).
		Int("loaded", len(keys)).
		Msg("loaded keys from supplier.yaml")

	return keys, nil
}

// parseHexKeyWithAddress parses a hex-encoded private key and derives the operator address.
func parseHexKeyWithAddress(hexKey string) (cryptotypes.PrivKey, string, error) {
	// Remove 0x prefix if present
	hexKey = strings.TrimPrefix(hexKey, "0x")
	hexKey = strings.TrimSpace(hexKey)

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid hex private key: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(keyBytes))
	}

	privKey := &secp256k1.PrivKey{Key: keyBytes}

	// Derive the operator address from the public key
	pubKey := privKey.PubKey()
	addr := cosmostypes.AccAddress(pubKey.Address())
	operatorAddr := addr.String()

	return privKey, operatorAddr, nil
}

// SupportsHotReload returns true if this provider supports hot-reload.
func (p *SupplierKeysFileProvider) SupportsHotReload() bool {
	return true
}

// WatchForChanges returns a channel that signals when keys may have changed.
func (p *SupplierKeysFileProvider) WatchForChanges(ctx context.Context) <-chan struct{} {
	// Start watching goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-p.watcher.Events:
				if !ok {
					return
				}
				// Trigger on Write or Create (file replacement)
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					// Non-blocking send
					select {
					case p.changeCh <- struct{}{}:
					default:
					}
				}
			case err, ok := <-p.watcher.Errors:
				if !ok {
					return
				}
				p.logger.Warn().Err(err).Msg("file watcher error")
			}
		}
	}()

	return p.changeCh
}

// Close gracefully shuts down the provider.
func (p *SupplierKeysFileProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if p.watcher != nil {
		p.watcher.Close()
	}

	close(p.changeCh)

	return nil
}
