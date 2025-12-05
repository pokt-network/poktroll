package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/keys"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	redistransport "github.com/pokt-network/poktroll/pkg/ha/transport/redis"
	"github.com/pokt-network/poktroll/pkg/ha/tx"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SupplierQueryClient queries supplier information from the blockchain.
type SupplierQueryClient interface {
	GetSupplier(ctx context.Context, supplierOperatorAddress string) (sharedtypes.Supplier, error)
}

// SupplierStatus represents the state of a supplier in the miner.
type SupplierStatus int

const (
	// SupplierStatusActive means the supplier is actively processing relays.
	SupplierStatusActive SupplierStatus = iota
	// SupplierStatusDraining means the supplier is being removed but waiting for pending work.
	SupplierStatusDraining
)

// SupplierState holds the state for a single supplier in the miner.
type SupplierState struct {
	OperatorAddr string
	Services     []string
	Status       SupplierStatus

	// Redis stream consumer for this supplier
	Consumer *redistransport.StreamsConsumer

	// Session management
	SessionStore    *RedisSessionStore
	WAL             *RedisWAL
	SnapshotManager *SMSTSnapshotManager

	// SMST management (for building and managing session trees)
	SMSTManager *InMemorySMSTManager

	// Lifecycle management (for claim/proof submission with timing spread)
	LifecycleManager  *SessionLifecycleManager
	LifecycleCallback *LifecycleCallback
	SupplierClient    *tx.HASupplierClient

	// Pending work tracking
	ActiveSessions int
	PendingClaims  int
	PendingProofs  int

	// Lifecycle
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// SupplierManagerConfig contains configuration for the SupplierManager.
type SupplierManagerConfig struct {
	// Redis connection
	RedisClient redis.UniversalClient

	// Stream configuration
	StreamPrefix  string
	ConsumerGroup string
	ConsumerName  string

	// Session configuration
	SessionTTL time.Duration

	// WAL configuration
	WALMaxLen int64

	// SupplierCache for publishing supplier state to relayers
	SupplierCache *cache.SupplierCache

	// MinerID identifies this miner instance (for debugging/tracking)
	MinerID string

	// SupplierQueryClient queries supplier information from the blockchain
	// Used to fetch the supplier's staked services
	SupplierQueryClient SupplierQueryClient

	// TxClient for submitting claims and proofs to the blockchain
	// This is a shared client for all suppliers
	TxClient *tx.TxClient

	// BlockClient for monitoring block heights (claim/proof timing)
	BlockClient client.BlockClient

	// SharedClient for querying shared parameters (claim/proof windows)
	SharedClient client.SharedQueryClient

	// SessionClient for querying session information
	SessionClient SessionQueryClient
}

// SupplierManager manages multiple suppliers in the HA Miner.
// It handles dynamic addition/removal of suppliers based on key changes.
type SupplierManager struct {
	logger     polylog.Logger
	config     SupplierManagerConfig
	keyManager keys.KeyManager
	registry   *SupplierRegistry

	// Per-supplier state
	suppliers   map[string]*SupplierState
	suppliersMu sync.RWMutex

	// Message processing callback
	onRelay func(ctx context.Context, supplierAddr string, msg *transport.StreamMessage) error

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	closed   bool
	mu       sync.Mutex
}

// NewSupplierManager creates a new supplier manager.
func NewSupplierManager(
	logger polylog.Logger,
	keyManager keys.KeyManager,
	registry *SupplierRegistry,
	config SupplierManagerConfig,
) *SupplierManager {
	return &SupplierManager{
		logger:     logging.ForComponent(logger, logging.ComponentSupplierManager),
		config:     config,
		keyManager: keyManager,
		registry:   registry,
		suppliers:  make(map[string]*SupplierState),
	}
}

// SetRelayHandler sets the callback for processing incoming relays.
func (m *SupplierManager) SetRelayHandler(handler func(ctx context.Context, supplierAddr string, msg *transport.StreamMessage) error) {
	m.onRelay = handler
}

// Start starts the supplier manager and begins processing.
func (m *SupplierManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("supplier manager is closed")
	}
	m.ctx, m.cancelFn = context.WithCancel(ctx)
	m.mu.Unlock()

	// Register for key changes
	m.keyManager.OnKeyChange(m.onKeyChange)

	// Initialize suppliers for all current keys
	for _, operatorAddr := range m.keyManager.ListSuppliers() {
		if err := m.addSupplier(m.ctx, operatorAddr); err != nil {
			m.logger.Warn().
				Err(err).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to add supplier on startup")
		}
	}

	m.logger.Info().
		Int("suppliers", len(m.suppliers)).
		Msg("supplier manager started")

	return nil
}

// onKeyChange handles key addition/removal notifications.
func (m *SupplierManager) onKeyChange(operatorAddr string, added bool) {
	if added {
		m.logger.Info().
			Str(logging.FieldSupplier, operatorAddr).
			Msg("key added, initializing supplier")

		if err := m.addSupplier(m.ctx, operatorAddr); err != nil {
			m.logger.Error().
				Err(err).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to add supplier")
		}
	} else {
		m.logger.Info().
			Str(logging.FieldSupplier, operatorAddr).
			Msg("key removed, draining supplier")

		go m.removeSupplier(operatorAddr)
	}
}

// addSupplier adds a new supplier to the manager.
func (m *SupplierManager) addSupplier(ctx context.Context, operatorAddr string) error {
	m.suppliersMu.Lock()
	defer m.suppliersMu.Unlock()

	// Check if already exists
	if _, exists := m.suppliers[operatorAddr]; exists {
		return nil // Already added
	}

	// Create supplier-specific context
	supplierCtx, cancelFn := context.WithCancel(ctx)

	// Create session store for this supplier
	sessionStore := NewRedisSessionStore(
		m.logger,
		m.config.RedisClient,
		SessionStoreConfig{
			KeyPrefix:       "ha:miner:sessions",
			SupplierAddress: operatorAddr,
			SessionTTL:      m.config.SessionTTL,
		},
	)

	// Create WAL for this supplier
	wal := NewRedisWAL(
		m.logger,
		m.config.RedisClient,
		WALConfig{
			SupplierAddress: operatorAddr,
			KeyPrefix:       "ha:miner:wal",
			MaxLen:          m.config.WALMaxLen,
		},
	)

	// Create SMST snapshot manager
	snapshotManager := NewSMSTSnapshotManager(
		m.logger,
		sessionStore,
		wal,
		SMSTRecoveryConfig{
			SupplierAddress: operatorAddr,
			RecoveryTimeout: 5 * time.Minute,
		},
	)

	// Create consumer for this supplier
	consumer, err := redistransport.NewStreamsConsumer(
		m.logger,
		m.config.RedisClient,
		transport.ConsumerConfig{
			StreamPrefix:            m.config.StreamPrefix,
			SupplierOperatorAddress: operatorAddr,
			ConsumerGroup:           m.config.ConsumerGroup,
			ConsumerName:            m.config.ConsumerName,
			BatchSize:               100,
			BlockTimeout:            5000,
			ClaimIdleTimeout:        30000,
			MaxRetries:              3,
		},
	)
	if err != nil {
		cancelFn()
		return fmt.Errorf("failed to create consumer for %s: %w", operatorAddr, err)
	}

	// Create SMST manager for building session trees
	smstManager := NewInMemorySMSTManager(
		m.logger,
		InMemorySMSTManagerConfig{
			SupplierAddress: operatorAddr,
		},
	)

	// Create supplier client for claim/proof submission
	var supplierClient *tx.HASupplierClient
	var lifecycleCallback *LifecycleCallback
	var lifecycleManager *SessionLifecycleManager

	// Only create lifecycle components if TxClient and BlockClient are provided
	if m.config.TxClient != nil && m.config.BlockClient != nil && m.config.SharedClient != nil {
		supplierClient = tx.NewHASupplierClient(
			m.config.TxClient,
			operatorAddr,
			m.logger,
		)

		// Create lifecycle callback for claim/proof submission
		lifecycleCallback = NewLifecycleCallback(
			m.logger,
			supplierClient,
			m.config.SharedClient,
			m.config.BlockClient,
			m.config.SessionClient,
			smstManager,
			snapshotManager,
			LifecycleCallbackConfig{
				SupplierAddress:    operatorAddr,
				ClaimRetryAttempts: 3,
				ClaimRetryDelay:    2 * time.Second,
				ProofRetryAttempts: 3,
				ProofRetryDelay:    2 * time.Second,
			},
		)

		// Create lifecycle manager for monitoring sessions and triggering claim/proof
		lifecycleManager = NewSessionLifecycleManager(
			m.logger,
			sessionStore,
			m.config.SharedClient,
			m.config.BlockClient,
			lifecycleCallback,
			SessionLifecycleConfig{
				SupplierAddress:          operatorAddr,
				CheckIntervalBlocks:      1,
				ClaimSubmissionBuffer:    2,
				ProofSubmissionBuffer:    2,
				MaxConcurrentTransitions: 10,
			},
		)

		// Start lifecycle manager
		if startErr := lifecycleManager.Start(supplierCtx); startErr != nil {
			m.logger.Warn().
				Err(startErr).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to start lifecycle manager, continuing without lifecycle management")
			lifecycleManager = nil
		} else {
			// Wire up callback so snapshot manager notifies lifecycle manager of new sessions
			// This is critical for tracking sessions created after startup
			lm := lifecycleManager // capture for closure
			snapshotManager.SetOnSessionCreatedCallback(func(ctx context.Context, snapshot *SessionSnapshot) error {
				return lm.TrackSession(ctx, snapshot)
			})
			m.logger.Info().
				Str(logging.FieldSupplier, operatorAddr).
				Msg("wired session creation callback to lifecycle manager")
		}
	} else {
		m.logger.Warn().
			Str(logging.FieldSupplier, operatorAddr).
			Msg("lifecycle management disabled - TxClient, BlockClient, or SharedClient not configured")
	}

	state := &SupplierState{
		OperatorAddr:      operatorAddr,
		Status:            SupplierStatusActive,
		Consumer:          consumer,
		SessionStore:      sessionStore,
		WAL:               wal,
		SnapshotManager:   snapshotManager,
		SMSTManager:       smstManager,
		LifecycleManager:  lifecycleManager,
		LifecycleCallback: lifecycleCallback,
		SupplierClient:    supplierClient,
		cancelFn:          cancelFn,
	}

	m.suppliers[operatorAddr] = state

	// Start consuming in background
	state.wg.Add(1)
	go m.consumeForSupplier(supplierCtx, state)

	// Publish to registry
	if m.registry != nil {
		if err := m.registry.PublishSupplierUpdate(ctx, SupplierUpdateActionAdd, operatorAddr, nil); err != nil {
			m.logger.Warn().
				Err(err).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to publish supplier add to registry")
		}
	}

	// Query supplier's staked services from the blockchain
	var services []string
	var ownerAddr string
	if m.config.SupplierQueryClient != nil {
		supplier, queryErr := m.config.SupplierQueryClient.GetSupplier(ctx, operatorAddr)
		if queryErr != nil {
			m.logger.Warn().
				Err(queryErr).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to query supplier from blockchain, will publish without services")
		} else {
			ownerAddr = supplier.OwnerAddress
			for _, svc := range supplier.Services {
				if svc != nil {
					services = append(services, svc.ServiceId)
				}
			}
			m.logger.Info().
				Str(logging.FieldSupplier, operatorAddr).
				Str("services", fmt.Sprintf("%v", services)).
				Msg("queried supplier services from blockchain")
		}
	}
	state.Services = services

	// Publish supplier state to cache for relayers to read
	if m.config.SupplierCache != nil {
		supplierState := &cache.SupplierState{
			Status:          cache.SupplierStatusActive,
			OperatorAddress: operatorAddr,
			OwnerAddress:    ownerAddr,
			Services:        services,
			UpdatedBy:       m.config.MinerID,
		}
		if cacheErr := m.config.SupplierCache.SetSupplierState(ctx, supplierState); cacheErr != nil {
			m.logger.Warn().
				Err(cacheErr).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to publish supplier state to cache")
		} else {
			m.logger.Info().
				Str(logging.FieldSupplier, operatorAddr).
				Str("services", fmt.Sprintf("%v", services)).
				Msg("published supplier state to cache")
		}
	}

	supplierManagerSuppliersActive.Inc()

	m.logger.Info().
		Str(logging.FieldSupplier, operatorAddr).
		Msg("supplier added and consuming")

	return nil
}

// consumeForSupplier runs the consume loop for a single supplier.
func (m *SupplierManager) consumeForSupplier(ctx context.Context, state *SupplierState) {
	defer state.wg.Done()

	msgChan := state.Consumer.Consume(ctx)

	for msg := range msgChan {
		// When draining, we continue processing existing messages
		// but log that we're in drain mode for visibility
		if state.Status == SupplierStatusDraining {
			m.logger.Debug().
				Str(logging.FieldSupplier, state.OperatorAddr).
				Msg("processing relay during drain")
		}

		// Process the relay
		if m.onRelay != nil {
			if err := m.onRelay(ctx, state.OperatorAddr, &msg); err != nil {
				m.logger.Warn().
					Err(err).
					Str(logging.FieldSupplier, state.OperatorAddr).
					Str("session_id", msg.Message.SessionId).
					Msg("failed to process relay")
				continue
			}
		}

		// Acknowledge
		if err := state.Consumer.Ack(ctx, msg.ID); err != nil {
			m.logger.Warn().
				Err(err).
				Str(logging.FieldSupplier, state.OperatorAddr).
				Msg("failed to acknowledge message")
		}
	}
}

// removeSupplier gracefully removes a supplier (waits for pending work).
func (m *SupplierManager) removeSupplier(operatorAddr string) {
	m.suppliersMu.Lock()
	state, exists := m.suppliers[operatorAddr]
	if !exists {
		m.suppliersMu.Unlock()
		return
	}

	// Mark as draining
	state.Status = SupplierStatusDraining
	m.suppliersMu.Unlock()

	// Publish draining status to registry
	if m.registry != nil {
		if err := m.registry.PublishSupplierUpdate(m.ctx, SupplierUpdateActionDraining, operatorAddr, nil); err != nil {
			m.logger.Warn().Err(err).Str(logging.FieldSupplier, operatorAddr).Msg("failed to publish draining status")
		}
	}

	// Update cache to mark supplier as unstaking
	if m.config.SupplierCache != nil {
		supplierState := &cache.SupplierState{
			Status:          cache.SupplierStatusUnstaking,
			OperatorAddress: operatorAddr,
			Services:        state.Services,
			UpdatedBy:       m.config.MinerID,
		}
		if cacheErr := m.config.SupplierCache.SetSupplierState(m.ctx, supplierState); cacheErr != nil {
			m.logger.Warn().
				Err(cacheErr).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to update supplier state to unstaking in cache")
		}
	}

	m.logger.Info().
		Str(logging.FieldSupplier, operatorAddr).
		Msg("supplier marked as draining, waiting for pending work...")

	// Wait for pending work (TODO: implement proper tracking)
	// For now, just wait for consumer to finish
	state.cancelFn()
	state.wg.Wait()

	// Cleanup
	m.suppliersMu.Lock()
	defer m.suppliersMu.Unlock()

	// Close lifecycle manager first to stop monitoring
	if state.LifecycleManager != nil {
		state.LifecycleManager.Close()
	}

	// Close SMST manager
	if state.SMSTManager != nil {
		state.SMSTManager.Close()
	}

	state.Consumer.Close()
	state.SnapshotManager.Close()
	state.WAL.Close()
	state.SessionStore.Close()

	delete(m.suppliers, operatorAddr)

	// Publish removal to registry
	if m.registry != nil {
		if err := m.registry.PublishSupplierUpdate(m.ctx, SupplierUpdateActionRemove, operatorAddr, nil); err != nil {
			m.logger.Warn().Err(err).Str(logging.FieldSupplier, operatorAddr).Msg("failed to publish removal status")
		}
	}

	// Delete supplier from cache
	if m.config.SupplierCache != nil {
		if cacheErr := m.config.SupplierCache.DeleteSupplierState(m.ctx, operatorAddr); cacheErr != nil {
			m.logger.Warn().
				Err(cacheErr).
				Str(logging.FieldSupplier, operatorAddr).
				Msg("failed to delete supplier state from cache")
		}
	}

	supplierManagerSuppliersActive.Dec()

	m.logger.Info().
		Str(logging.FieldSupplier, operatorAddr).
		Msg("supplier gracefully removed")
}

// GetSupplierState returns the state for a specific supplier.
func (m *SupplierManager) GetSupplierState(operatorAddr string) (*SupplierState, bool) {
	m.suppliersMu.RLock()
	defer m.suppliersMu.RUnlock()

	state, ok := m.suppliers[operatorAddr]
	return state, ok
}

// ListSuppliers returns all active supplier addresses.
func (m *SupplierManager) ListSuppliers() []string {
	m.suppliersMu.RLock()
	defer m.suppliersMu.RUnlock()

	suppliers := make([]string, 0, len(m.suppliers))
	for addr := range m.suppliers {
		suppliers = append(suppliers, addr)
	}
	return suppliers
}

// Close gracefully shuts down the supplier manager.
func (m *SupplierManager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true

	if m.cancelFn != nil {
		m.cancelFn()
	}
	m.mu.Unlock()

	// Wait for all suppliers to finish
	m.suppliersMu.Lock()
	for _, state := range m.suppliers {
		state.cancelFn()
		state.wg.Wait()

		// Close lifecycle manager first
		if state.LifecycleManager != nil {
			state.LifecycleManager.Close()
		}

		// Close SMST manager
		if state.SMSTManager != nil {
			state.SMSTManager.Close()
		}

		state.Consumer.Close()
		state.SnapshotManager.Close()
		state.WAL.Close()
		state.SessionStore.Close()
	}
	m.suppliers = make(map[string]*SupplierState)
	m.suppliersMu.Unlock()

	m.logger.Info().Msg("supplier manager closed")
	return nil
}
