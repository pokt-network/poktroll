package keys

import (
	"context"
	"fmt"
	"sync"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ KeyManager = (*MultiProviderKeyManager)(nil)

// MultiProviderKeyManager implements KeyManager using multiple KeyProviders.
// It aggregates keys from all providers and supports hot-reload.
type MultiProviderKeyManager struct {
	logger    polylog.Logger
	providers []KeyProvider
	config    KeyManagerConfig

	// Keys storage
	keys   map[string]cryptotypes.PrivKey // operatorAddr -> privKey
	keysMu sync.RWMutex

	// Change callbacks
	callbacks   []KeyChangeCallback
	callbacksMu sync.RWMutex

	// Lifecycle
	mu       sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewMultiProviderKeyManager creates a new KeyManager with multiple providers.
func NewMultiProviderKeyManager(
	logger polylog.Logger,
	providers []KeyProvider,
	config KeyManagerConfig,
) *MultiProviderKeyManager {
	return &MultiProviderKeyManager{
		logger:    logging.ForComponent(logger, logging.ComponentKeyManager),
		providers: providers,
		config:    config,
		keys:      make(map[string]cryptotypes.PrivKey),
	}
}

// Start starts background processes (file watching, etc.)
func (m *MultiProviderKeyManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("key manager is closed")
	}

	ctx, m.cancelFn = context.WithCancel(ctx)
	m.mu.Unlock()

	// Initial key load
	if err := m.Reload(ctx); err != nil {
		return fmt.Errorf("failed to load initial keys: %w", err)
	}

	// Start watching each provider that supports hot-reload
	if m.config.HotReloadEnabled {
		for _, provider := range m.providers {
			if provider.SupportsHotReload() {
				m.wg.Add(1)
				go m.watchProvider(ctx, provider)
			}
		}
	}

	m.logger.Info().
		Int("providers", len(m.providers)).
		Int("keys", len(m.keys)).
		Msg("key manager started")

	return nil
}

// watchProvider watches a single provider for key changes.
func (m *MultiProviderKeyManager) watchProvider(ctx context.Context, provider KeyProvider) {
	defer m.wg.Done()

	changes := provider.WatchForChanges(ctx)
	if changes == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			m.logger.Info().
				Str("provider", provider.Name()).
				Msg("key change detected, reloading")

			if err := m.Reload(ctx); err != nil {
				m.logger.Error().
					Err(err).
					Str("provider", provider.Name()).
					Msg("failed to reload keys")
			}
		}
	}
}

// GetSigner returns the private key for signing for a given operator address.
func (m *MultiProviderKeyManager) GetSigner(operatorAddr string) (cryptotypes.PrivKey, error) {
	m.keysMu.RLock()
	defer m.keysMu.RUnlock()

	key, ok := m.keys[operatorAddr]
	if !ok {
		return nil, fmt.Errorf("no key found for operator %s", operatorAddr)
	}

	return key, nil
}

// ListSuppliers returns all operator addresses that have signing keys.
func (m *MultiProviderKeyManager) ListSuppliers() []string {
	m.keysMu.RLock()
	defer m.keysMu.RUnlock()

	suppliers := make([]string, 0, len(m.keys))
	for addr := range m.keys {
		suppliers = append(suppliers, addr)
	}
	return suppliers
}

// HasKey returns true if a key exists for the given operator address.
func (m *MultiProviderKeyManager) HasKey(operatorAddr string) bool {
	m.keysMu.RLock()
	defer m.keysMu.RUnlock()

	_, ok := m.keys[operatorAddr]
	return ok
}

// AddKey dynamically adds a new supplier key.
func (m *MultiProviderKeyManager) AddKey(operatorAddr string, key cryptotypes.PrivKey) error {
	m.keysMu.Lock()
	_, existed := m.keys[operatorAddr]
	m.keys[operatorAddr] = key
	m.keysMu.Unlock()

	if !existed {
		m.notifyKeyChange(operatorAddr, true)
	}

	m.logger.Info().
		Str("operator", operatorAddr).
		Bool("replaced", existed).
		Msg("added key")

	return nil
}

// RemoveKey removes a supplier key.
func (m *MultiProviderKeyManager) RemoveKey(operatorAddr string) error {
	m.keysMu.Lock()
	_, existed := m.keys[operatorAddr]
	if !existed {
		m.keysMu.Unlock()
		return fmt.Errorf("no key found for operator %s", operatorAddr)
	}
	delete(m.keys, operatorAddr)
	m.keysMu.Unlock()

	m.notifyKeyChange(operatorAddr, false)

	m.logger.Info().
		Str("operator", operatorAddr).
		Msg("removed key")

	return nil
}

// Reload reloads keys from all configured sources.
func (m *MultiProviderKeyManager) Reload(ctx context.Context) error {
	newKeys := make(map[string]cryptotypes.PrivKey)

	// Load keys from each provider
	for _, provider := range m.providers {
		keys, err := provider.LoadKeys(ctx)
		if err != nil {
			m.logger.Warn().
				Err(err).
				Str("provider", provider.Name()).
				Msg("failed to load keys from provider")
			continue
		}

		for addr, key := range keys {
			if _, exists := newKeys[addr]; exists {
				m.logger.Warn().
					Str("operator", addr).
					Str("provider", provider.Name()).
					Msg("duplicate key, using later provider")
			}
			newKeys[addr] = key
		}

		m.logger.Debug().
			Str("provider", provider.Name()).
			Int("keys", len(keys)).
			Msg("loaded keys from provider")
	}

	// Determine added and removed keys
	m.keysMu.Lock()
	oldKeys := m.keys

	added := make([]string, 0)
	removed := make([]string, 0)

	// Find added keys
	for addr := range newKeys {
		if _, existed := oldKeys[addr]; !existed {
			added = append(added, addr)
		}
	}

	// Find removed keys
	for addr := range oldKeys {
		if _, exists := newKeys[addr]; !exists {
			removed = append(removed, addr)
		}
	}

	m.keys = newKeys
	m.keysMu.Unlock()

	// Notify callbacks
	for _, addr := range added {
		m.notifyKeyChange(addr, true)
	}
	for _, addr := range removed {
		m.notifyKeyChange(addr, false)
	}

	m.logger.Info().
		Int("total", len(newKeys)).
		Int("added", len(added)).
		Int("removed", len(removed)).
		Msg("reloaded keys")

	keyReloadsTotal.Inc()
	supplierKeysActive.Set(float64(len(newKeys)))

	return nil
}

// notifyKeyChange notifies all callbacks of a key change.
func (m *MultiProviderKeyManager) notifyKeyChange(operatorAddr string, added bool) {
	m.callbacksMu.RLock()
	callbacks := make([]KeyChangeCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.callbacksMu.RUnlock()

	for _, cb := range callbacks {
		cb(operatorAddr, added)
	}

	if added {
		keyChangesTotal.WithLabelValues("added").Inc()
	} else {
		keyChangesTotal.WithLabelValues("removed").Inc()
	}
}

// OnKeyChange registers a callback that is called when keys change.
func (m *MultiProviderKeyManager) OnKeyChange(callback KeyChangeCallback) {
	m.callbacksMu.Lock()
	defer m.callbacksMu.Unlock()

	m.callbacks = append(m.callbacks, callback)
}

// Close gracefully shuts down the key manager.
func (m *MultiProviderKeyManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	if m.cancelFn != nil {
		m.cancelFn()
	}

	// Close all providers
	for _, provider := range m.providers {
		if err := provider.Close(); err != nil {
			m.logger.Warn().
				Err(err).
				Str("provider", provider.Name()).
				Msg("error closing provider")
		}
	}

	m.wg.Wait()

	m.logger.Info().Msg("key manager closed")
	return nil
}
