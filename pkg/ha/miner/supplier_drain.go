package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// SupplierDrainConfig contains configuration for supplier draining.
type SupplierDrainConfig struct {
	// DrainTimeout is the maximum time to wait for a supplier to drain.
	DrainTimeout time.Duration

	// CheckInterval is how often to check drain progress.
	CheckInterval time.Duration
}

// DefaultSupplierDrainConfig returns sensible defaults.
func DefaultSupplierDrainConfig() SupplierDrainConfig {
	return SupplierDrainConfig{
		DrainTimeout:  30 * time.Minute,
		CheckInterval: 5 * time.Second,
	}
}

// DrainState represents the current state of a supplier drain operation.
type DrainState string

const (
	// DrainStateNotStarted means no drain has been initiated.
	DrainStateNotStarted DrainState = "not_started"

	// DrainStateDraining means the supplier is being drained.
	DrainStateDraining DrainState = "draining"

	// DrainStateCompleted means the drain completed successfully.
	DrainStateCompleted DrainState = "completed"

	// DrainStateFailed means the drain failed (timeout or error).
	DrainStateFailed DrainState = "failed"
)

// SupplierDrainStatus tracks the drain status for a supplier.
type SupplierDrainStatus struct {
	// SupplierAddress is the supplier being drained.
	SupplierAddress string

	// State is the current drain state.
	State DrainState

	// StartedAt is when the drain started.
	StartedAt time.Time

	// CompletedAt is when the drain completed (if completed).
	CompletedAt time.Time

	// PendingSessions is the number of sessions still pending.
	PendingSessions int

	// PendingClaims is the number of claims still pending.
	PendingClaims int

	// PendingProofs is the number of proofs still pending.
	PendingProofs int

	// Error contains any error that occurred.
	Error error
}

// SessionTracker provides session tracking for drain operations.
type SessionTracker interface {
	// GetPendingSessionCount returns the count of sessions pending settlement.
	GetPendingSessionCount() int

	// GetSessionsByState returns all sessions in a given state.
	GetSessionsByState(state SessionState) []*SessionSnapshot

	// HasPendingSessions returns true if there are sessions not yet settled.
	HasPendingSessions() bool
}

// SupplierDrainManager manages graceful supplier removal.
type SupplierDrainManager struct {
	logger polylog.Logger
	config SupplierDrainConfig

	// Drain status per supplier
	drainStatus   map[string]*SupplierDrainStatus
	drainStatusMu sync.RWMutex

	// Session trackers per supplier
	sessionTrackers   map[string]SessionTracker
	sessionTrackersMu sync.RWMutex

	// Drain completion callbacks
	onDrainComplete   func(supplierAddr string)
	onDrainCompleteMu sync.RWMutex

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
}

// NewSupplierDrainManager creates a new drain manager.
func NewSupplierDrainManager(
	logger polylog.Logger,
	config SupplierDrainConfig,
) *SupplierDrainManager {
	if config.DrainTimeout == 0 {
		config.DrainTimeout = 30 * time.Minute
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 5 * time.Second
	}

	return &SupplierDrainManager{
		logger:          logging.ForComponent(logger, logging.ComponentSupplierDrain),
		config:          config,
		drainStatus:     make(map[string]*SupplierDrainStatus),
		sessionTrackers: make(map[string]SessionTracker),
	}
}

// Start begins the drain manager.
func (m *SupplierDrainManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("drain manager is closed")
	}

	m.ctx, m.cancelFn = context.WithCancel(ctx)
	m.mu.Unlock()

	m.logger.Info().Msg("supplier drain manager started")
	return nil
}

// RegisterSessionTracker registers a session tracker for a supplier.
func (m *SupplierDrainManager) RegisterSessionTracker(supplierAddr string, tracker SessionTracker) {
	m.sessionTrackersMu.Lock()
	m.sessionTrackers[supplierAddr] = tracker
	m.sessionTrackersMu.Unlock()
}

// UnregisterSessionTracker removes a session tracker.
func (m *SupplierDrainManager) UnregisterSessionTracker(supplierAddr string) {
	m.sessionTrackersMu.Lock()
	delete(m.sessionTrackers, supplierAddr)
	m.sessionTrackersMu.Unlock()
}

// SetDrainCompleteCallback sets the callback for when a drain completes.
func (m *SupplierDrainManager) SetDrainCompleteCallback(callback func(supplierAddr string)) {
	m.onDrainCompleteMu.Lock()
	m.onDrainComplete = callback
	m.onDrainCompleteMu.Unlock()
}

// InitiateDrain starts draining a supplier.
// It marks the supplier as draining and begins monitoring for completion.
func (m *SupplierDrainManager) InitiateDrain(ctx context.Context, supplierAddr string) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("drain manager is closed")
	}
	m.mu.RUnlock()

	// Check if already draining
	m.drainStatusMu.Lock()
	if status, exists := m.drainStatus[supplierAddr]; exists {
		if status.State == DrainStateDraining {
			m.drainStatusMu.Unlock()
			return fmt.Errorf("supplier %s is already draining", supplierAddr)
		}
	}

	// Create drain status
	status := &SupplierDrainStatus{
		SupplierAddress: supplierAddr,
		State:           DrainStateDraining,
		StartedAt:       time.Now(),
	}
	m.drainStatus[supplierAddr] = status
	m.drainStatusMu.Unlock()

	m.logger.Info().
		Str("supplier", supplierAddr).
		Msg("initiated supplier drain")

	// Start drain monitor
	m.wg.Add(1)
	go m.monitorDrain(ctx, supplierAddr)

	return nil
}

// monitorDrain monitors a supplier drain until completion or timeout.
func (m *SupplierDrainManager) monitorDrain(ctx context.Context, supplierAddr string) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	timeout := time.After(m.config.DrainTimeout)

	for {
		select {
		case <-ctx.Done():
			m.updateDrainStatus(supplierAddr, DrainStateFailed, ctx.Err())
			return

		case <-m.ctx.Done():
			m.updateDrainStatus(supplierAddr, DrainStateFailed, m.ctx.Err())
			return

		case <-timeout:
			m.logger.Warn().
				Str("supplier", supplierAddr).
				Dur("timeout", m.config.DrainTimeout).
				Msg("supplier drain timed out")
			m.updateDrainStatus(supplierAddr, DrainStateFailed, fmt.Errorf("drain timeout"))
			return

		case <-ticker.C:
			if m.checkDrainComplete(supplierAddr) {
				m.logger.Info().
					Str("supplier", supplierAddr).
					Msg("supplier drain completed successfully")
				m.updateDrainStatus(supplierAddr, DrainStateCompleted, nil)
				m.notifyDrainComplete(supplierAddr)
				return
			}
		}
	}
}

// checkDrainComplete checks if a supplier has finished draining.
func (m *SupplierDrainManager) checkDrainComplete(supplierAddr string) bool {
	m.sessionTrackersMu.RLock()
	tracker, exists := m.sessionTrackers[supplierAddr]
	m.sessionTrackersMu.RUnlock()

	if !exists {
		// No tracker, consider drained
		return true
	}

	// Update status with current counts
	m.drainStatusMu.Lock()
	status := m.drainStatus[supplierAddr]
	if status != nil {
		status.PendingSessions = tracker.GetPendingSessionCount()

		claimingSessions := tracker.GetSessionsByState(SessionStateClaiming)
		claimedSessions := tracker.GetSessionsByState(SessionStateClaimed)
		provingSessions := tracker.GetSessionsByState(SessionStateProving)

		status.PendingClaims = len(claimingSessions) + len(claimedSessions)
		status.PendingProofs = len(provingSessions)
	}
	m.drainStatusMu.Unlock()

	// Check if all sessions are settled
	return !tracker.HasPendingSessions()
}

// updateDrainStatus updates the drain status for a supplier.
func (m *SupplierDrainManager) updateDrainStatus(supplierAddr string, state DrainState, err error) {
	m.drainStatusMu.Lock()
	defer m.drainStatusMu.Unlock()

	status, exists := m.drainStatus[supplierAddr]
	if !exists {
		return
	}

	status.State = state
	status.Error = err

	if state == DrainStateCompleted || state == DrainStateFailed {
		status.CompletedAt = time.Now()
	}
}

// notifyDrainComplete calls the drain complete callback.
func (m *SupplierDrainManager) notifyDrainComplete(supplierAddr string) {
	m.onDrainCompleteMu.RLock()
	callback := m.onDrainComplete
	m.onDrainCompleteMu.RUnlock()

	if callback != nil {
		callback(supplierAddr)
	}
}

// GetDrainStatus returns the drain status for a supplier.
func (m *SupplierDrainManager) GetDrainStatus(supplierAddr string) *SupplierDrainStatus {
	m.drainStatusMu.RLock()
	defer m.drainStatusMu.RUnlock()

	if status, exists := m.drainStatus[supplierAddr]; exists {
		// Return a copy
		copy := *status
		return &copy
	}
	return nil
}

// GetAllDrainStatuses returns all drain statuses.
func (m *SupplierDrainManager) GetAllDrainStatuses() []*SupplierDrainStatus {
	m.drainStatusMu.RLock()
	defer m.drainStatusMu.RUnlock()

	result := make([]*SupplierDrainStatus, 0, len(m.drainStatus))
	for _, status := range m.drainStatus {
		copy := *status
		result = append(result, &copy)
	}
	return result
}

// IsDraining returns true if the supplier is currently draining.
func (m *SupplierDrainManager) IsDraining(supplierAddr string) bool {
	m.drainStatusMu.RLock()
	defer m.drainStatusMu.RUnlock()

	if status, exists := m.drainStatus[supplierAddr]; exists {
		return status.State == DrainStateDraining
	}
	return false
}

// CancelDrain cancels a drain operation.
func (m *SupplierDrainManager) CancelDrain(supplierAddr string) {
	m.drainStatusMu.Lock()
	defer m.drainStatusMu.Unlock()

	if status, exists := m.drainStatus[supplierAddr]; exists {
		if status.State == DrainStateDraining {
			status.State = DrainStateNotStarted
			status.CompletedAt = time.Now()
			m.logger.Info().
				Str("supplier", supplierAddr).
				Msg("drain cancelled")
		}
	}
}

// WaitForDrain waits for a supplier drain to complete.
func (m *SupplierDrainManager) WaitForDrain(ctx context.Context, supplierAddr string) error {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status := m.GetDrainStatus(supplierAddr)
			if status == nil {
				return nil // Not draining
			}

			switch status.State {
			case DrainStateCompleted:
				return nil
			case DrainStateFailed:
				return status.Error
			case DrainStateNotStarted:
				return nil
			}
		}
	}
}

// Close gracefully shuts down the drain manager.
func (m *SupplierDrainManager) Close() error {
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

	m.logger.Info().Msg("supplier drain manager closed")
	return nil
}

// GracefulRemover provides a higher-level API for graceful supplier removal.
type GracefulRemover struct {
	logger       polylog.Logger
	drainManager *SupplierDrainManager

	// Callbacks
	onSupplierRemoved func(supplierAddr string, success bool)
}

// NewGracefulRemover creates a new graceful remover.
func NewGracefulRemover(
	logger polylog.Logger,
	drainManager *SupplierDrainManager,
) *GracefulRemover {
	return &GracefulRemover{
		logger:       logging.ForComponent(logger, "graceful_remover"),
		drainManager: drainManager,
	}
}

// SetSupplierRemovedCallback sets the callback for when a supplier is removed.
func (r *GracefulRemover) SetSupplierRemovedCallback(callback func(supplierAddr string, success bool)) {
	r.onSupplierRemoved = callback
}

// RemoveSupplier initiates graceful removal of a supplier.
// It waits for all pending claims/proofs to complete before returning.
func (r *GracefulRemover) RemoveSupplier(ctx context.Context, supplierAddr string) error {
	r.logger.Info().
		Str("supplier", supplierAddr).
		Msg("starting graceful supplier removal")

	// Initiate drain
	if err := r.drainManager.InitiateDrain(ctx, supplierAddr); err != nil {
		return fmt.Errorf("failed to initiate drain: %w", err)
	}

	// Wait for drain to complete
	err := r.drainManager.WaitForDrain(ctx, supplierAddr)

	success := err == nil
	if r.onSupplierRemoved != nil {
		r.onSupplierRemoved(supplierAddr, success)
	}

	if err != nil {
		r.logger.Error().
			Err(err).
			Str("supplier", supplierAddr).
			Msg("graceful removal failed")
		return err
	}

	r.logger.Info().
		Str("supplier", supplierAddr).
		Msg("graceful supplier removal completed")

	return nil
}

// RemoveSupplierAsync initiates graceful removal without waiting.
func (r *GracefulRemover) RemoveSupplierAsync(ctx context.Context, supplierAddr string) error {
	r.logger.Info().
		Str("supplier", supplierAddr).
		Msg("starting async graceful supplier removal")

	if err := r.drainManager.InitiateDrain(ctx, supplierAddr); err != nil {
		return fmt.Errorf("failed to initiate drain: %w", err)
	}

	// Set up callback for completion
	r.drainManager.SetDrainCompleteCallback(func(addr string) {
		if addr == supplierAddr && r.onSupplierRemoved != nil {
			r.onSupplierRemoved(addr, true)
		}
	})

	return nil
}
