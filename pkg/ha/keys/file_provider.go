package keys

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ KeyProvider = (*FileKeyProvider)(nil)

// FileKeyProvider loads keys from YAML/JSON files in a directory.
// It supports hot-reload via fsnotify.
type FileKeyProvider struct {
	logger   polylog.Logger
	keysDir  string
	watcher  *fsnotify.Watcher
	changeCh chan struct{}

	mu     sync.Mutex
	closed bool
}

// KeyFile is the structure of a key file.
type KeyFile struct {
	// OperatorAddress is the bech32 operator address.
	OperatorAddress string `yaml:"operator_address" json:"operator_address"`

	// PrivateKeyHex is the hex-encoded secp256k1 private key.
	// Can be prefixed with "0x" or not.
	PrivateKeyHex string `yaml:"private_key_hex" json:"private_key_hex"`
}

// NewFileKeyProvider creates a new file-based key provider.
func NewFileKeyProvider(logger polylog.Logger, keysDir string) (*FileKeyProvider, error) {
	// Verify directory exists
	info, err := os.Stat(keysDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the directory
			if mkdirErr := os.MkdirAll(keysDir, 0700); mkdirErr != nil {
				return nil, fmt.Errorf("failed to create keys directory: %w", mkdirErr)
			}
		} else {
			return nil, fmt.Errorf("failed to stat keys directory: %w", err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("keys path is not a directory: %s", keysDir)
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(keysDir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch keys directory: %w", err)
	}

	return &FileKeyProvider{
		logger:   logging.ForComponent(logger, logging.ComponentKeyFileProvider),
		keysDir:  keysDir,
		watcher:  watcher,
		changeCh: make(chan struct{}, 1),
	}, nil
}

// Name returns a human-readable name for this provider.
func (p *FileKeyProvider) Name() string {
	return "file:" + p.keysDir
}

// LoadKeys loads all keys from YAML/JSON files in the directory.
func (p *FileKeyProvider) LoadKeys(ctx context.Context) (map[string]cryptotypes.PrivKey, error) {
	keys := make(map[string]cryptotypes.PrivKey)

	entries, err := os.ReadDir(p.keysDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(p.keysDir, name)
		key, operatorAddr, err := p.loadKeyFile(path)
		if err != nil {
			p.logger.Warn().
				Err(err).
				Str("file", name).
				Msg("failed to load key file")
			keyLoadErrors.WithLabelValues("file").Inc()
			continue
		}

		keys[operatorAddr] = key
		p.logger.Debug().
			Str("file", name).
			Str("operator", operatorAddr).
			Msg("loaded key from file")
	}

	return keys, nil
}

// loadKeyFile loads a single key file.
func (p *FileKeyProvider) loadKeyFile(path string) (cryptotypes.PrivKey, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	var keyFile KeyFile
	if unmarshalErr := yaml.Unmarshal(data, &keyFile); unmarshalErr != nil {
		return nil, "", fmt.Errorf("failed to parse file: %w", unmarshalErr)
	}

	if keyFile.OperatorAddress == "" {
		return nil, "", fmt.Errorf("missing operator_address")
	}

	if keyFile.PrivateKeyHex == "" {
		return nil, "", fmt.Errorf("missing private_key_hex")
	}

	// Parse hex-encoded private key
	hexKey := strings.TrimPrefix(keyFile.PrivateKeyHex, "0x")
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid hex private key: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(keyBytes))
	}

	privKey := &secp256k1.PrivKey{Key: keyBytes}

	return privKey, keyFile.OperatorAddress, nil
}

// SupportsHotReload returns true if this provider supports hot-reload.
func (p *FileKeyProvider) SupportsHotReload() bool {
	return true
}

// WatchForChanges returns a channel that signals when keys may have changed.
func (p *FileKeyProvider) WatchForChanges(ctx context.Context) <-chan struct{} {
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
				// Only trigger on Create, Write, Remove
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove) != 0 {
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
func (p *FileKeyProvider) Close() error {
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
