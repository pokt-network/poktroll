package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// SMSTRecoveryConfig contains configuration for SMST recovery.
type SMSTRecoveryConfig struct {
	// SupplierAddress is the supplier this recovery service is for.
	SupplierAddress string

	// RecoveryTimeout is the maximum time allowed for recovery.
	RecoveryTimeout time.Duration
}

// RecoveredSession contains the information needed to rebuild an SMST.
type RecoveredSession struct {
	// Snapshot contains the session metadata.
	Snapshot *SessionSnapshot

	// RelayUpdates contains the relay updates to replay.
	// Each update is a WALEntry containing RelayHash, RelayBytes, and ComputeUnits.
	RelayUpdates []*WALEntry
}

// SMSTRecoveryService provides session recovery capabilities for HA failover.
// When a new leader takes over, it uses this service to:
// 1. Load active session snapshots from Redis
// 2. Replay WAL entries to get the relay updates
// 3. Provide the data needed to rebuild SMST trees
type SMSTRecoveryService struct {
	logger       polylog.Logger
	sessionStore SessionStore
	wal          WAL
	config       SMSTRecoveryConfig

	mu       sync.Mutex
	closed   bool
	sessions map[string]*RecoveredSession
}

// NewSMSTRecoveryService creates a new SMST recovery service.
func NewSMSTRecoveryService(
	logger polylog.Logger,
	sessionStore SessionStore,
	wal WAL,
	config SMSTRecoveryConfig,
) *SMSTRecoveryService {
	if config.RecoveryTimeout == 0 {
		config.RecoveryTimeout = 5 * time.Minute
	}

	return &SMSTRecoveryService{
		logger:       logging.ForSupplierComponent(logger, logging.ComponentSMSTRecovery, config.SupplierAddress),
		sessionStore: sessionStore,
		wal:          wal,
		config:       config,
		sessions:     make(map[string]*RecoveredSession),
	}
}

// RecoverSessions recovers all active sessions for the supplier.
// This should be called when a new leader is elected.
// Returns the list of recovered sessions with their relay updates.
func (s *SMSTRecoveryService) RecoverSessions(ctx context.Context) ([]*RecoveredSession, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("recovery service is closed")
	}
	s.mu.Unlock()

	startTime := time.Now()
	defer func() {
		smstRecoveryLatency.WithLabelValues(s.config.SupplierAddress).Observe(time.Since(startTime).Seconds())
	}()

	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, s.config.RecoveryTimeout)
	defer cancel()

	s.logger.Info().Msg("starting session recovery for HA failover")

	// Get all sessions that need recovery (active, claiming, claimed, proving)
	statesToRecover := []SessionState{
		SessionStateActive,
		SessionStateClaiming,
		SessionStateClaimed,
		SessionStateProving,
	}

	var allSnapshots []*SessionSnapshot
	for _, state := range statesToRecover {
		snapshots, err := s.sessionStore.GetByState(ctx, s.config.SupplierAddress, state)
		if err != nil {
			return nil, fmt.Errorf("failed to get sessions in state %s: %w", state, err)
		}
		allSnapshots = append(allSnapshots, snapshots...)
	}

	if len(allSnapshots) == 0 {
		s.logger.Info().Msg("no sessions to recover")
		return nil, nil
	}

	s.logger.Info().
		Int("session_count", len(allSnapshots)).
		Msg("found sessions to recover")

	// Recover each session
	recoveredSessions := make([]*RecoveredSession, 0, len(allSnapshots))
	for _, snapshot := range allSnapshots {
		recovered, err := s.recoverSession(ctx, snapshot)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("session_id", snapshot.SessionID).
				Msg("failed to recover session, skipping")
			continue
		}

		recoveredSessions = append(recoveredSessions, recovered)
		s.sessions[snapshot.SessionID] = recovered

		smstRecoveries.WithLabelValues(s.config.SupplierAddress, snapshot.SessionID).Inc()
	}

	s.logger.Info().
		Int("recovered_count", len(recoveredSessions)).
		Int("total_sessions", len(allSnapshots)).
		Dur("duration", time.Since(startTime)).
		Msg("session recovery completed")

	return recoveredSessions, nil
}

// recoverSession recovers a single session by replaying its WAL entries.
func (s *SMSTRecoveryService) recoverSession(ctx context.Context, snapshot *SessionSnapshot) (*RecoveredSession, error) {
	logger := s.logger.With(
		"session_id", snapshot.SessionID,
		"state", string(snapshot.State),
	)

	logger.Debug().Msg("recovering session")

	// Get the checkpoint to know where to start replaying from
	checkpoint, err := s.wal.GetCheckpoint(ctx, snapshot.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get WAL checkpoint: %w", err)
	}

	// Determine where to start reading from
	startFrom := ""
	if checkpoint != "" && checkpoint != "0" {
		startFrom = checkpoint
		logger.Debug().
			Str("checkpoint", checkpoint).
			Msg("replaying WAL from checkpoint")
	} else {
		logger.Debug().Msg("replaying WAL from beginning (no checkpoint)")
	}

	// Read all WAL entries for this session from the checkpoint
	entries, err := s.wal.ReadFrom(ctx, snapshot.SessionID, startFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL entries: %w", err)
	}

	logger.Debug().
		Int("entry_count", len(entries)).
		Msg("recovered WAL entries")

	return &RecoveredSession{
		Snapshot:     snapshot,
		RelayUpdates: entries,
	}, nil
}

// RecoverSession recovers a single session by ID.
func (s *SMSTRecoveryService) RecoverSession(ctx context.Context, sessionID string) (*RecoveredSession, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("recovery service is closed")
	}

	// Check if already recovered
	if session, ok := s.sessions[sessionID]; ok {
		s.mu.Unlock()
		return session, nil
	}
	s.mu.Unlock()

	// Get the session snapshot
	snapshot, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session snapshot: %w", err)
	}
	if snapshot == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	recovered, err := s.recoverSession(ctx, snapshot)
	if err != nil {
		return nil, err
	}

	// Cache the recovered session
	s.mu.Lock()
	s.sessions[sessionID] = recovered
	s.mu.Unlock()

	return recovered, nil
}

// GetRecoveredSession returns a previously recovered session.
func (s *SMSTRecoveryService) GetRecoveredSession(sessionID string) (*RecoveredSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	return session, ok
}

// ClearRecoveredSession removes a session from the recovered sessions cache.
// This should be called after the session has been successfully rebuilt.
func (s *SMSTRecoveryService) ClearRecoveredSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
}

// Close shuts down the recovery service.
func (s *SMSTRecoveryService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.logger.Info().Msg("SMST recovery service closed")
	return nil
}

// SessionCreatedCallback is called when a new session is created.
// This allows external components (like SessionLifecycleManager) to be notified.
type SessionCreatedCallback func(ctx context.Context, snapshot *SessionSnapshot) error

// SMSTSnapshotManager manages the lifecycle of SMST snapshots.
// It coordinates between the SessionStore, WAL, and recovery service.
type SMSTSnapshotManager struct {
	logger       polylog.Logger
	sessionStore SessionStore
	wal          WAL
	recovery     *SMSTRecoveryService
	config       SMSTRecoveryConfig

	// onSessionCreated is called when a new session is created
	onSessionCreated SessionCreatedCallback

	mu     sync.Mutex
	closed bool
}

// NewSMSTSnapshotManager creates a new SMST snapshot manager.
func NewSMSTSnapshotManager(
	logger polylog.Logger,
	sessionStore SessionStore,
	wal WAL,
	config SMSTRecoveryConfig,
) *SMSTSnapshotManager {
	return &SMSTSnapshotManager{
		logger:       logging.ForSupplierComponent(logger, logging.ComponentSMSTSnapshot, config.SupplierAddress),
		sessionStore: sessionStore,
		wal:          wal,
		recovery:     NewSMSTRecoveryService(logger, sessionStore, wal, config),
		config:       config,
	}
}

// SetOnSessionCreatedCallback sets the callback to be invoked when a new session is created.
// This allows external components (like SessionLifecycleManager) to track new sessions.
func (m *SMSTSnapshotManager) SetOnSessionCreatedCallback(callback SessionCreatedCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSessionCreated = callback
}

// OnRelayMined should be called when a relay is mined and added to an SMST.
// It logs the relay update to the WAL and updates the session snapshot.
// If the session doesn't exist, it will be created automatically.
func (m *SMSTSnapshotManager) OnRelayMined(
	ctx context.Context,
	sessionID string,
	relayHash, relayBytes []byte,
	computeUnits uint64,
) error {
	return m.OnRelayMinedWithMetadata(ctx, sessionID, relayHash, relayBytes, computeUnits, "", "", "", 0, 0)
}

// OnRelayMinedWithMetadata should be called when a relay is mined and added to an SMST.
// It logs the relay update to the WAL and updates the session snapshot.
// If the session doesn't exist, it will be created automatically using the provided metadata.
func (m *SMSTSnapshotManager) OnRelayMinedWithMetadata(
	ctx context.Context,
	sessionID string,
	relayHash, relayBytes []byte,
	computeUnits uint64,
	supplierAddress, serviceID, applicationAddress string,
	sessionStartHeight, sessionEndHeight int64,
) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("snapshot manager is closed")
	}
	m.mu.Unlock()

	// Check if session exists, create if not
	snapshot, err := m.sessionStore.Get(ctx, sessionID)
	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("failed to check session existence")
	}

	if snapshot == nil {
		// Session doesn't exist, create it
		if supplierAddress == "" || serviceID == "" {
			m.logger.Warn().
				Str("session_id", sessionID).
				Msg("session not found and missing metadata to create it")
		} else {
			m.logger.Info().
				Str("session_id", sessionID).
				Str("service_id", serviceID).
				Str("supplier", supplierAddress).
				Int64("start_height", sessionStartHeight).
				Int64("end_height", sessionEndHeight).
				Msg("creating new session on first relay")

			if err := m.OnSessionCreated(ctx, sessionID, supplierAddress, serviceID, applicationAddress, sessionStartHeight, sessionEndHeight); err != nil {
				m.logger.Warn().
					Err(err).
					Str("session_id", sessionID).
					Msg("failed to create session")
			} else {
				// Record session creation metric for operator visibility
				RecordSessionCreated(supplierAddress, serviceID)
			}
		}
	}

	// Create WAL entry
	entry := &WALEntry{
		SessionID:    sessionID,
		RelayHash:    relayHash,
		RelayBytes:   relayBytes,
		ComputeUnits: computeUnits,
	}

	// Append to WAL
	entryID, err := m.wal.Append(ctx, sessionID, entry)
	if err != nil {
		return fmt.Errorf("failed to append to WAL: %w", err)
	}

	// Update session snapshot
	if err := m.sessionStore.IncrementRelayCount(ctx, sessionID, computeUnits); err != nil {
		m.logger.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("failed to update session relay count")
	}

	if err := m.sessionStore.UpdateWALPosition(ctx, sessionID, entryID); err != nil {
		m.logger.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("failed to update session WAL position")
	}

	return nil
}

// OnSessionCreated should be called when a new session is created.
func (m *SMSTSnapshotManager) OnSessionCreated(
	ctx context.Context,
	sessionID string,
	supplierAddress string,
	serviceID string,
	applicationAddress string,
	startHeight, endHeight int64,
) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("snapshot manager is closed")
	}
	// Grab the callback while holding the lock
	callback := m.onSessionCreated
	m.mu.Unlock()

	snapshot := &SessionSnapshot{
		SessionID:               sessionID,
		SupplierOperatorAddress: supplierAddress,
		ServiceID:               serviceID,
		ApplicationAddress:      applicationAddress,
		SessionStartHeight:      startHeight,
		SessionEndHeight:        endHeight,
		State:                   SessionStateActive,
	}

	if err := m.sessionStore.Save(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save session snapshot: %w", err)
	}

	m.logger.Debug().
		Str("session_id", sessionID).
		Msg("session created and snapshot saved")

	// Notify the lifecycle manager (if configured) so it can track this session
	if callback != nil {
		if err := callback(ctx, snapshot); err != nil {
			m.logger.Warn().
				Err(err).
				Str("session_id", sessionID).
				Msg("failed to notify lifecycle manager of new session")
			// Don't return error - session was saved successfully
		}
	}

	return nil
}

// OnSessionStateChange should be called when a session's state changes.
func (m *SMSTSnapshotManager) OnSessionStateChange(
	ctx context.Context,
	sessionID string,
	newState SessionState,
) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("snapshot manager is closed")
	}
	m.mu.Unlock()

	if err := m.sessionStore.UpdateState(ctx, sessionID, newState); err != nil {
		return fmt.Errorf("failed to update session state: %w", err)
	}

	// If the session is claimed, create a checkpoint
	if newState == SessionStateClaimed {
		if err := m.createCheckpoint(ctx, sessionID); err != nil {
			m.logger.Warn().
				Err(err).
				Str("session_id", sessionID).
				Msg("failed to create checkpoint on claim")
		}
	}

	return nil
}

// OnSessionClaimed should be called when a session is claimed.
// It stores the claim root hash and creates a WAL checkpoint.
func (m *SMSTSnapshotManager) OnSessionClaimed(
	ctx context.Context,
	sessionID string,
	claimRootHash []byte,
) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("snapshot manager is closed")
	}
	m.mu.Unlock()

	// Get current snapshot
	snapshot, err := m.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session snapshot: %w", err)
	}
	if snapshot == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Update with claim root hash
	snapshot.ClaimedRootHash = claimRootHash
	snapshot.State = SessionStateClaimed

	if err := m.sessionStore.Save(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save session snapshot: %w", err)
	}

	// Create a checkpoint
	if err := m.createCheckpoint(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}

	m.logger.Debug().
		Str("session_id", sessionID).
		Int("root_hash_len", len(claimRootHash)).
		Msg("session claimed and checkpoint created")

	return nil
}

// OnSessionSettled should be called when a session is settled.
// It cleans up the WAL entries and optionally the session snapshot.
func (m *SMSTSnapshotManager) OnSessionSettled(
	ctx context.Context,
	sessionID string,
) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("snapshot manager is closed")
	}
	m.mu.Unlock()

	// Update state to settled
	if err := m.sessionStore.UpdateState(ctx, sessionID, SessionStateSettled); err != nil {
		m.logger.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("failed to update session state to settled")
	}

	// Delete WAL entries for this session
	if deleter, ok := m.wal.(*RedisWAL); ok {
		if err := deleter.DeleteSession(ctx, sessionID); err != nil {
			m.logger.Warn().
				Err(err).
				Str("session_id", sessionID).
				Msg("failed to delete WAL entries")
		}
	}

	m.logger.Debug().
		Str("session_id", sessionID).
		Msg("session settled and WAL cleaned up")

	return nil
}

// createCheckpoint creates a WAL checkpoint for the session.
func (m *SMSTSnapshotManager) createCheckpoint(ctx context.Context, sessionID string) error {
	// Get the last WAL entry ID
	snapshot, err := m.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if snapshot == nil || snapshot.LastWALEntryID == "" {
		return nil // No WAL entries to checkpoint
	}

	if err := m.wal.Checkpoint(ctx, sessionID, snapshot.LastWALEntryID); err != nil {
		return fmt.Errorf("failed to create WAL checkpoint: %w", err)
	}

	return nil
}

// RecoverSessions delegates to the recovery service.
func (m *SMSTSnapshotManager) RecoverSessions(ctx context.Context) ([]*RecoveredSession, error) {
	return m.recovery.RecoverSessions(ctx)
}

// Close shuts down the snapshot manager.
func (m *SMSTSnapshotManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	if err := m.recovery.Close(); err != nil {
		m.logger.Warn().Err(err).Msg("failed to close recovery service")
	}

	m.logger.Info().Msg("SMST snapshot manager closed")
	return nil
}
