package relayer

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// RelayMeterConfig contains configuration for the relay meter.
type RelayMeterConfig struct {
	// OverServicingEnabled allows suppliers to serve beyond app stake limits.
	OverServicingEnabled bool

	// RedisKeyPrefix is the prefix for Redis keys.
	RedisKeyPrefix string

	// SessionCleanupInterval is how often to clean up expired session meters.
	SessionCleanupInterval time.Duration
}

// DefaultRelayMeterConfig returns sensible defaults.
func DefaultRelayMeterConfig() RelayMeterConfig {
	return RelayMeterConfig{
		OverServicingEnabled:   true,
		RedisKeyPrefix:         "ha:relay_meter",
		SessionCleanupInterval: 30 * time.Second,
	}
}

// SessionMeterState represents the metering state for a session.
type SessionMeterState struct {
	// SessionID is the unique session identifier.
	SessionID string

	// AppAddress is the application address.
	AppAddress string

	// ServiceID is the service being consumed.
	ServiceID string

	// MaxStake is the maximum stake this app can consume with this supplier.
	MaxStake cosmostypes.Coin

	// ConsumedStake is the amount already consumed.
	ConsumedStake cosmostypes.Coin

	// OverServicedRelays counts relays served beyond the limit.
	OverServicedRelays uint64

	// SessionEndHeight is when the session ends.
	SessionEndHeight int64

	// LastUpdated is when this state was last modified.
	LastUpdated time.Time
}

// RelayMeter manages rate limiting based on application stake.
type RelayMeter struct {
	logger        polylog.Logger
	config        RelayMeterConfig
	redisClient   redis.UniversalClient
	appClient     client.ApplicationQueryClient
	sharedClient  client.SharedQueryClient
	sessionClient client.SessionQueryClient
	blockClient   client.BlockClient

	// Local cache of session meters (L1)
	sessionMeters   map[string]*SessionMeterState
	sessionMetersMu sync.RWMutex

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
}

// NewRelayMeter creates a new relay meter.
func NewRelayMeter(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	appClient client.ApplicationQueryClient,
	sharedClient client.SharedQueryClient,
	sessionClient client.SessionQueryClient,
	blockClient client.BlockClient,
	config RelayMeterConfig,
) *RelayMeter {
	if config.RedisKeyPrefix == "" {
		config.RedisKeyPrefix = "ha:relay_meter"
	}
	if config.SessionCleanupInterval == 0 {
		config.SessionCleanupInterval = 30 * time.Second
	}

	return &RelayMeter{
		logger:        logging.ForComponent(logger, logging.ComponentRelayMeter),
		config:        config,
		redisClient:   redisClient,
		appClient:     appClient,
		sharedClient:  sharedClient,
		sessionClient: sessionClient,
		blockClient:   blockClient,
		sessionMeters: make(map[string]*SessionMeterState),
	}
}

// Start begins the relay meter background processes.
func (m *RelayMeter) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("relay meter is closed")
	}

	m.ctx, m.cancelFn = context.WithCancel(ctx)
	m.mu.Unlock()

	// Start cleanup worker
	m.wg.Add(1)
	go m.cleanupWorker(m.ctx)

	m.logger.Info().
		Bool("over_servicing_enabled", m.config.OverServicingEnabled).
		Msg("relay meter started")

	return nil
}

// CheckAndConsumeRelay checks if a relay can be served and consumes stake if so.
// Returns:
// - allowed: true if the relay should be served
// - overServiced: true if this relay exceeds the app's stake limit
// - err: any error that occurred
func (m *RelayMeter) CheckAndConsumeRelay(
	ctx context.Context,
	sessionID string,
	appAddress string,
	serviceID string,
	sessionEndHeight int64,
) (allowed bool, overServiced bool, err error) {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return false, false, fmt.Errorf("relay meter is closed")
	}
	m.mu.RUnlock()

	// Get or create session meter
	meter, err := m.getOrCreateSessionMeter(ctx, sessionID, appAddress, serviceID, sessionEndHeight)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get session meter, allowing relay")
		return true, false, nil
	}

	// Calculate relay cost
	relayCost, err := m.calculateRelayCost(ctx, serviceID)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to calculate relay cost, allowing relay")
		return true, false, nil
	}

	// Check if within limits
	newConsumed := meter.ConsumedStake.Add(relayCost)

	m.sessionMetersMu.Lock()
	defer m.sessionMetersMu.Unlock()

	if meter.MaxStake.IsGTE(newConsumed) {
		// Within limits
		meter.ConsumedStake = newConsumed
		meter.LastUpdated = time.Now()

		relayMeterConsumptions.WithLabelValues(serviceID, "within_limit").Inc()
		return true, false, nil
	}

	// Over the limit
	meter.OverServicedRelays++
	meter.LastUpdated = time.Now()

	relayMeterConsumptions.WithLabelValues(serviceID, "over_limit").Inc()

	if m.config.OverServicingEnabled {
		// Log at power-of-2 intervals
		if shouldLogOverServicing(meter.OverServicedRelays) {
			m.logger.Warn().
				Str("application", appAddress).
				Str(logging.FieldSessionID, sessionID).
				Uint64("over_serviced_count", meter.OverServicedRelays).
				Msg("application over-serviced (over-servicing enabled)")
		}
		return true, true, nil
	}

	m.logger.Debug().
		Str("application", appAddress).
		Str(logging.FieldSessionID, sessionID).
		Msg("relay rejected due to stake limit")

	return false, true, nil
}

// RevertRelayConsumption reverts the stake consumption for a relay that wasn't mined.
func (m *RelayMeter) RevertRelayConsumption(
	ctx context.Context,
	sessionID string,
	serviceID string,
) error {
	m.sessionMetersMu.Lock()
	defer m.sessionMetersMu.Unlock()

	meter, exists := m.sessionMeters[sessionID]
	if !exists {
		return nil // No meter, nothing to revert
	}

	relayCost, err := m.calculateRelayCost(ctx, serviceID)
	if err != nil {
		return nil // Can't calculate, skip revert
	}

	if meter.ConsumedStake.IsGTE(relayCost) {
		meter.ConsumedStake = meter.ConsumedStake.Sub(relayCost)
		meter.LastUpdated = time.Now()
	}

	return nil
}

// AllowOverServicing returns whether over-servicing is enabled.
func (m *RelayMeter) AllowOverServicing() bool {
	return m.config.OverServicingEnabled
}

// GetSessionMeterState returns the current meter state for a session.
func (m *RelayMeter) GetSessionMeterState(sessionID string) *SessionMeterState {
	m.sessionMetersMu.RLock()
	defer m.sessionMetersMu.RUnlock()

	if meter, exists := m.sessionMeters[sessionID]; exists {
		// Return a copy
		copy := *meter
		return &copy
	}
	return nil
}

// getOrCreateSessionMeter gets or creates a session meter.
func (m *RelayMeter) getOrCreateSessionMeter(
	ctx context.Context,
	sessionID string,
	appAddress string,
	serviceID string,
	sessionEndHeight int64,
) (*SessionMeterState, error) {
	// Check L1 cache first
	m.sessionMetersMu.RLock()
	if meter, exists := m.sessionMeters[sessionID]; exists {
		m.sessionMetersMu.RUnlock()
		return meter, nil
	}
	m.sessionMetersMu.RUnlock()

	// Create new meter
	app, err := m.appClient.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	sharedParams, err := m.sharedClient.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared params: %w", err)
	}

	sessionParams, err := m.sessionClient.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get session params: %w", err)
	}

	// Calculate max stake for this session/supplier
	maxStake := calculateAppStakePerSessionSupplier(
		app.GetStake(),
		sharedParams,
		sessionParams.GetNumSuppliersPerSession(),
	)

	meter := &SessionMeterState{
		SessionID:          sessionID,
		AppAddress:         appAddress,
		ServiceID:          serviceID,
		MaxStake:           maxStake,
		ConsumedStake:      cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
		OverServicedRelays: 0,
		SessionEndHeight:   sessionEndHeight,
		LastUpdated:        time.Now(),
	}

	m.sessionMetersMu.Lock()
	// Double-check another goroutine didn't create it
	if existing, exists := m.sessionMeters[sessionID]; exists {
		m.sessionMetersMu.Unlock()
		return existing, nil
	}
	m.sessionMeters[sessionID] = meter
	m.sessionMetersMu.Unlock()

	relayMeterSessionsActive.Inc()

	return meter, nil
}

// calculateRelayCost calculates the cost of a single relay in uPOKT.
func (m *RelayMeter) calculateRelayCost(ctx context.Context, serviceID string) (cosmostypes.Coin, error) {
	sharedParams, err := m.sharedClient.GetParams(ctx)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// For now, use a default compute units per relay
	// In production, this should query the service configuration
	computeUnitsPerRelay := uint64(1)

	computeUnitCostUpokt := new(big.Rat).SetFrac64(
		int64(sharedParams.GetComputeUnitsToTokensMultiplier()),
		int64(sharedParams.GetComputeUnitCostGranularity()),
	)

	relayCostRat := new(big.Rat).Mul(
		new(big.Rat).SetUint64(computeUnitsPerRelay),
		computeUnitCostUpokt,
	)

	estimatedRelayCost := big.NewInt(0).Quo(relayCostRat.Num(), relayCostRat.Denom())
	return cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewIntFromBigInt(estimatedRelayCost)), nil
}

// cleanupWorker periodically cleans up expired session meters.
func (m *RelayMeter) cleanupWorker(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.SessionCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanupExpiredSessions(ctx)
		}
	}
}

// cleanupExpiredSessions removes session meters for sessions past claim window.
func (m *RelayMeter) cleanupExpiredSessions(ctx context.Context) {
	sharedParams, err := m.sharedClient.GetParams(ctx)
	if err != nil {
		return
	}

	block := m.blockClient.LastBlock(ctx)
	currentHeight := block.Height()

	// Find sessions to delete
	m.sessionMetersMu.RLock()
	var toDelete []string
	for sessionID, meter := range m.sessionMeters {
		claimWindowOpen := sharedtypes.GetClaimWindowOpenHeight(sharedParams, meter.SessionEndHeight)
		if currentHeight >= claimWindowOpen {
			toDelete = append(toDelete, sessionID)
		}
	}
	m.sessionMetersMu.RUnlock()

	if len(toDelete) == 0 {
		return
	}

	// Delete expired sessions
	m.sessionMetersMu.Lock()
	for _, sessionID := range toDelete {
		delete(m.sessionMeters, sessionID)
		relayMeterSessionsActive.Dec()
	}
	m.sessionMetersMu.Unlock()

	m.logger.Debug().
		Int("cleaned_up", len(toDelete)).
		Msg("cleaned up expired session meters")
}

// Close gracefully shuts down the relay meter.
func (m *RelayMeter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	if m.cancelFn != nil {
		m.cancelFn()
	}

	m.wg.Wait()

	m.logger.Info().Msg("relay meter closed")
	return nil
}

// calculateAppStakePerSessionSupplier calculates the portion of app stake
// available to a single supplier in a single session.
func calculateAppStakePerSessionSupplier(
	stake *cosmostypes.Coin,
	sharedParams *sharedtypes.Params,
	numSuppliersPerSession uint64,
) cosmostypes.Coin {
	// Split among suppliers in the session
	appStakePerSupplier := stake.Amount.Quo(cosmosmath.NewInt(int64(numSuppliersPerSession)))

	// Account for pending sessions
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	numBlocksUntilProofWindowCloses := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	numClosedSessionsAwaitingSettlement := math.Ceil(float64(numBlocksUntilProofWindowCloses) / float64(numBlocksPerSession))

	// Add 1 for current session
	pendingSessions := int64(numClosedSessionsAwaitingSettlement) + 1

	appStakePerSessionSupplier := appStakePerSupplier.Quo(cosmosmath.NewInt(pendingSessions))
	return cosmostypes.NewCoin(pocket.DenomuPOKT, appStakePerSessionSupplier)
}

// shouldLogOverServicing returns true if the occurrence count is a power of 2.
// This provides exponential backoff for logging.
func shouldLogOverServicing(occurrence uint64) bool {
	return (occurrence & (occurrence - 1)) == 0
}

// RelayMeterSnapshot captures the current state for monitoring/debugging.
type RelayMeterSnapshot struct {
	ActiveSessions       int
	TotalOverServiced    uint64
	OverServicingEnabled bool
}

// GetSnapshot returns a snapshot of the relay meter state.
func (m *RelayMeter) GetSnapshot() RelayMeterSnapshot {
	m.sessionMetersMu.RLock()
	defer m.sessionMetersMu.RUnlock()

	var totalOverServiced uint64
	for _, meter := range m.sessionMeters {
		totalOverServiced += meter.OverServicedRelays
	}

	return RelayMeterSnapshot{
		ActiveSessions:       len(m.sessionMeters),
		TotalOverServiced:    totalOverServiced,
		OverServicingEnabled: m.config.OverServicingEnabled,
	}
}
