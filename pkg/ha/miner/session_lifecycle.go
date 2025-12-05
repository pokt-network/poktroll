package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SessionLifecycleConfig contains configuration for the session lifecycle manager.
type SessionLifecycleConfig struct {
	// SupplierAddress is the supplier this manager is for.
	SupplierAddress string

	// CheckIntervalBlocks is how often to check for session transitions.
	// Default: 1 (every block)
	CheckIntervalBlocks int64

	// ClaimSubmissionBuffer is blocks before window close to start claiming.
	// This provides buffer time for transaction confirmation.
	// Default: 2
	ClaimSubmissionBuffer int64

	// ProofSubmissionBuffer is blocks before window close to start proving.
	// Default: 2
	ProofSubmissionBuffer int64

	// MaxConcurrentTransitions is the max number of sessions transitioning at once.
	// Default: 10
	MaxConcurrentTransitions int
}

// DefaultSessionLifecycleConfig returns sensible defaults.
func DefaultSessionLifecycleConfig() SessionLifecycleConfig {
	return SessionLifecycleConfig{
		CheckIntervalBlocks:      1,
		ClaimSubmissionBuffer:    2,
		ProofSubmissionBuffer:    2,
		MaxConcurrentTransitions: 10,
	}
}

// SessionLifecycleCallback defines callbacks for lifecycle events.
type SessionLifecycleCallback interface {
	// OnSessionActive is called when a new session starts.
	OnSessionActive(ctx context.Context, snapshot *SessionSnapshot) error

	// OnSessionNeedsClaim is called when a session needs a claim submitted.
	// The callback should trigger claim submission and return the root hash.
	OnSessionNeedsClaim(ctx context.Context, snapshot *SessionSnapshot) (rootHash []byte, err error)

	// OnSessionNeedsProof is called when a session needs a proof submitted.
	OnSessionNeedsProof(ctx context.Context, snapshot *SessionSnapshot) error

	// OnSessionSettled is called when a session is fully settled.
	OnSessionSettled(ctx context.Context, snapshot *SessionSnapshot) error

	// OnSessionExpired is called when a session expires without settling.
	OnSessionExpired(ctx context.Context, snapshot *SessionSnapshot, reason string) error
}

// SessionLifecycleManager manages the lifecycle of sessions from active to settled.
// It monitors block heights and triggers state transitions at the appropriate times.
type SessionLifecycleManager struct {
	logger       polylog.Logger
	config       SessionLifecycleConfig
	sessionStore SessionStore
	sharedClient client.SharedQueryClient
	blockClient  client.BlockClient
	callback     SessionLifecycleCallback

	// Current shared params (cached)
	sharedParams   *sharedtypes.Params
	sharedParamsMu sync.RWMutex

	// Active sessions being monitored
	activeSessions   map[string]*SessionSnapshot // sessionID -> snapshot
	activeSessionsMu sync.RWMutex

	// Transition semaphore to limit concurrent transitions
	transitionSem chan struct{}

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
}

// NewSessionLifecycleManager creates a new session lifecycle manager.
func NewSessionLifecycleManager(
	logger polylog.Logger,
	sessionStore SessionStore,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	callback SessionLifecycleCallback,
	config SessionLifecycleConfig,
) *SessionLifecycleManager {
	if config.CheckIntervalBlocks <= 0 {
		config.CheckIntervalBlocks = 1
	}
	if config.ClaimSubmissionBuffer <= 0 {
		config.ClaimSubmissionBuffer = 2
	}
	if config.ProofSubmissionBuffer <= 0 {
		config.ProofSubmissionBuffer = 2
	}
	if config.MaxConcurrentTransitions <= 0 {
		config.MaxConcurrentTransitions = 10
	}

	return &SessionLifecycleManager{
		logger:         logging.ForSupplierComponent(logger, logging.ComponentSessionLifecycle, config.SupplierAddress),
		config:         config,
		sessionStore:   sessionStore,
		sharedClient:   sharedClient,
		blockClient:    blockClient,
		callback:       callback,
		activeSessions: make(map[string]*SessionSnapshot),
		transitionSem:  make(chan struct{}, config.MaxConcurrentTransitions),
	}
}

// Start begins monitoring sessions and triggering lifecycle transitions.
func (m *SessionLifecycleManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("lifecycle manager is closed")
	}

	m.ctx, m.cancelFn = context.WithCancel(ctx)
	m.mu.Unlock()

	// Load initial shared params
	if err := m.refreshSharedParams(ctx); err != nil {
		return fmt.Errorf("failed to load shared params: %w", err)
	}

	// Load existing sessions from store
	if err := m.loadExistingSessions(ctx); err != nil {
		m.logger.Warn().Err(err).Msg("failed to load existing sessions, starting fresh")
	}

	// Start lifecycle checker
	m.wg.Add(1)
	go m.lifecycleChecker(m.ctx)

	m.logger.Info().
		Int("active_sessions", len(m.activeSessions)).
		Msg("session lifecycle manager started")

	return nil
}

// loadExistingSessions loads sessions from the store on startup.
func (m *SessionLifecycleManager) loadExistingSessions(ctx context.Context) error {
	sessions, err := m.sessionStore.GetBySupplier(ctx, m.config.SupplierAddress)
	if err != nil {
		return err
	}

	m.activeSessionsMu.Lock()
	defer m.activeSessionsMu.Unlock()

	for _, session := range sessions {
		// Only track sessions that aren't settled or expired
		if session.State != SessionStateSettled && session.State != SessionStateExpired {
			m.activeSessions[session.SessionID] = session
			sessionSnapshotsLoaded.WithLabelValues(m.config.SupplierAddress).Inc()
		} else {
			// Log and track metrics for skipped sessions (settled or expired)
			sessionSnapshotsSkippedAtStartup.WithLabelValues(m.config.SupplierAddress, string(session.State)).Inc()
			m.logger.Warn().
				Str("session_id", session.SessionID).
				Str("state", string(session.State)).
				Str("service_id", session.ServiceID).
				Int64("session_end_height", session.SessionEndHeight).
				Int64("relay_count", session.RelayCount).
				Msg("skipping session at startup (already settled or expired)")
		}
	}

	return nil
}

// refreshSharedParams refreshes the cached shared params.
func (m *SessionLifecycleManager) refreshSharedParams(ctx context.Context) error {
	params, err := m.sharedClient.GetParams(ctx)
	if err != nil {
		return err
	}

	m.sharedParamsMu.Lock()
	m.sharedParams = params
	m.sharedParamsMu.Unlock()

	return nil
}

// getSharedParams returns the cached shared params.
func (m *SessionLifecycleManager) getSharedParams() *sharedtypes.Params {
	m.sharedParamsMu.RLock()
	defer m.sharedParamsMu.RUnlock()
	return m.sharedParams
}

// TrackSession starts tracking a new session.
func (m *SessionLifecycleManager) TrackSession(ctx context.Context, snapshot *SessionSnapshot) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("lifecycle manager is closed")
	}
	m.mu.RUnlock()

	m.activeSessionsMu.Lock()
	m.activeSessions[snapshot.SessionID] = snapshot
	m.activeSessionsMu.Unlock()

	// Persist to store
	if err := m.sessionStore.Save(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	m.logger.Debug().
		Str("session_id", snapshot.SessionID).
		Str("state", string(snapshot.State)).
		Msg("started tracking session")

	return nil
}

// GetSession returns a tracked session by ID.
func (m *SessionLifecycleManager) GetSession(sessionID string) *SessionSnapshot {
	m.activeSessionsMu.RLock()
	defer m.activeSessionsMu.RUnlock()
	return m.activeSessions[sessionID]
}

// GetActiveSessions returns all sessions in the active state.
func (m *SessionLifecycleManager) GetActiveSessions() []*SessionSnapshot {
	m.activeSessionsMu.RLock()
	defer m.activeSessionsMu.RUnlock()

	result := make([]*SessionSnapshot, 0)
	for _, session := range m.activeSessions {
		if session.State == SessionStateActive {
			result = append(result, session)
		}
	}
	return result
}

// GetSessionsByState returns all sessions in a given state.
func (m *SessionLifecycleManager) GetSessionsByState(state SessionState) []*SessionSnapshot {
	m.activeSessionsMu.RLock()
	defer m.activeSessionsMu.RUnlock()

	result := make([]*SessionSnapshot, 0)
	for _, session := range m.activeSessions {
		if session.State == state {
			result = append(result, session)
		}
	}
	return result
}

// UpdateSessionRelayCount updates the relay count for a session.
func (m *SessionLifecycleManager) UpdateSessionRelayCount(ctx context.Context, sessionID string, computeUnits uint64) error {
	m.activeSessionsMu.Lock()
	session, exists := m.activeSessions[sessionID]
	if !exists {
		m.activeSessionsMu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.RelayCount++
	session.TotalComputeUnits += computeUnits
	session.LastUpdatedAt = time.Now()
	m.activeSessionsMu.Unlock()

	// Persist update asynchronously
	go func() {
		if err := m.sessionStore.IncrementRelayCount(ctx, sessionID, computeUnits); err != nil {
			m.logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to persist relay count")
		}
	}()

	return nil
}

// lifecycleChecker periodically checks sessions for state transitions.
func (m *SessionLifecycleManager) lifecycleChecker(ctx context.Context) {
	defer m.wg.Done()

	// Get block time for ticker interval
	// TODO: In the future, this could be derived from shared params
	// For now, use the standard 6 second block time
	_ = m.getSharedParams() // Ensure params are cached
	blockTime := 6 * time.Second

	ticker := time.NewTicker(blockTime)
	defer ticker.Stop()

	lastHeight := int64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get current block height
			block := m.blockClient.LastBlock(ctx)
			currentHeight := block.Height()

			// Only process if height changed
			if currentHeight <= lastHeight {
				continue
			}
			lastHeight = currentHeight
			currentBlockHeight.Set(float64(currentHeight))

			// Refresh shared params periodically (every 10 blocks)
			if currentHeight%10 == 0 {
				if err := m.refreshSharedParams(ctx); err != nil {
					m.logger.Warn().Err(err).Msg("failed to refresh shared params")
				}
			}

			// Check all sessions for transitions
			m.checkSessionTransitions(ctx, currentHeight)
		}
	}
}

// checkSessionTransitions checks all active sessions for required state transitions.
func (m *SessionLifecycleManager) checkSessionTransitions(ctx context.Context, currentHeight int64) {
	m.activeSessionsMu.RLock()
	sessions := make([]*SessionSnapshot, 0, len(m.activeSessions))
	for _, session := range m.activeSessions {
		sessions = append(sessions, session)
	}
	m.activeSessionsMu.RUnlock()

	params := m.getSharedParams()
	if params == nil {
		m.logger.Warn().Msg("shared params not available, skipping transition check")
		return
	}

	for _, session := range sessions {
		// Check if this session needs a transition
		newState, action := m.determineTransition(session, currentHeight, params)
		if newState == "" || newState == session.State {
			continue
		}

		// Acquire semaphore for transition
		select {
		case m.transitionSem <- struct{}{}:
			go func(s *SessionSnapshot, ns SessionState, act string) {
				defer func() { <-m.transitionSem }()
				m.executeTransition(ctx, s, ns, act)
			}(session, newState, action)
		default:
			// Too many concurrent transitions, skip for now
			m.logger.Debug().
				Str("session_id", session.SessionID).
				Msg("skipping transition, too many concurrent")
		}
	}
}

// determineTransition determines if a session needs to transition.
func (m *SessionLifecycleManager) determineTransition(
	session *SessionSnapshot,
	currentHeight int64,
	params *sharedtypes.Params,
) (newState SessionState, action string) {
	switch session.State {
	case SessionStateActive:
		// Check if session ended and claim window is approaching
		claimWindowOpen := sharedtypes.GetClaimWindowOpenHeight(params, session.SessionEndHeight)
		claimWindowClose := sharedtypes.GetClaimWindowCloseHeight(params, session.SessionEndHeight)

		// If we're in the claim window, transition to claiming
		if currentHeight >= claimWindowOpen && currentHeight < claimWindowClose-m.config.ClaimSubmissionBuffer {
			return SessionStateClaiming, "claim_window_open"
		}

		// If claim window has passed without claiming, session expired
		if currentHeight >= claimWindowClose {
			return SessionStateExpired, "claim_window_missed"
		}

	case SessionStateClaiming:
		// Transition to claimed happens after callback succeeds
		claimWindowClose := sharedtypes.GetClaimWindowCloseHeight(params, session.SessionEndHeight)

		// If claim window passed without submitting, expired
		if currentHeight >= claimWindowClose {
			return SessionStateExpired, "claim_failed"
		}

	case SessionStateClaimed:
		// Check if proof window has opened
		proofWindowOpen := sharedtypes.GetProofWindowOpenHeight(params, session.SessionEndHeight)
		proofWindowClose := sharedtypes.GetProofWindowCloseHeight(params, session.SessionEndHeight)

		// If we're in the proof window, transition to proving
		if currentHeight >= proofWindowOpen && currentHeight < proofWindowClose-m.config.ProofSubmissionBuffer {
			return SessionStateProving, "proof_window_open"
		}

		// If proof window passed without proving, session may still settle
		// (proof is optional if not selected for proof requirement)
		if currentHeight >= proofWindowClose {
			return SessionStateSettled, "proof_window_passed"
		}

	case SessionStateProving:
		// Transition to settled happens after callback succeeds
		proofWindowClose := sharedtypes.GetProofWindowCloseHeight(params, session.SessionEndHeight)

		// If proof window passed, check if settled
		if currentHeight >= proofWindowClose {
			return SessionStateSettled, "proof_submitted"
		}
	}

	return "", ""
}

// executeTransition executes a state transition for a session.
func (m *SessionLifecycleManager) executeTransition(
	ctx context.Context,
	session *SessionSnapshot,
	newState SessionState,
	action string,
) {
	oldState := session.State

	// Create session-scoped logger for this transition
	sessionLogger := logging.WithSession(m.logger, session.SessionID)

	sessionLogger.Info().
		Str(logging.FieldOldState, string(oldState)).
		Str(logging.FieldNewState, string(newState)).
		Str(logging.FieldAction, action).
		Msg("executing session transition")

	var err error

	switch newState {
	case SessionStateClaiming:
		// Call the claim callback
		rootHash, claimErr := m.callback.OnSessionNeedsClaim(ctx, session)
		if claimErr != nil {
			sessionLogger.Error().Err(claimErr).Msg("claim callback failed")
			claimErrors.WithLabelValues(m.config.SupplierAddress, "callback_failed").Inc()
			return
		}

		// Update session with root hash and transition to claimed
		session.ClaimedRootHash = rootHash
		newState = SessionStateClaimed

	case SessionStateProving:
		// Call the proof callback
		if proofErr := m.callback.OnSessionNeedsProof(ctx, session); proofErr != nil {
			sessionLogger.Error().Err(proofErr).Msg("proof callback failed")
			proofErrors.WithLabelValues(m.config.SupplierAddress, "callback_failed").Inc()
			return
		}

		// Transition to settled after successful proof
		newState = SessionStateSettled

	case SessionStateSettled:
		if settleErr := m.callback.OnSessionSettled(ctx, session); settleErr != nil {
			sessionLogger.Warn().Err(settleErr).Msg("settle callback failed")
		}

	case SessionStateExpired:
		if expireErr := m.callback.OnSessionExpired(ctx, session, action); expireErr != nil {
			sessionLogger.Warn().Err(expireErr).Msg("expire callback failed")
		}
	}

	// Update session state
	m.activeSessionsMu.Lock()
	session.State = newState
	session.LastUpdatedAt = time.Now()
	m.activeSessionsMu.Unlock()

	// Persist the state change
	err = m.sessionStore.UpdateState(ctx, session.SessionID, newState)
	if err != nil {
		sessionLogger.Error().Err(err).Msg("failed to persist state change")
		sessionStoreErrors.WithLabelValues(m.config.SupplierAddress, "update_state").Inc()
		return
	}

	// Record the transition
	sessionStateTransitions.WithLabelValues(
		m.config.SupplierAddress,
		string(oldState),
		string(newState),
	).Inc()

	// Remove settled/expired sessions from active tracking
	if newState == SessionStateSettled || newState == SessionStateExpired {
		m.activeSessionsMu.Lock()
		delete(m.activeSessions, session.SessionID)
		m.activeSessionsMu.Unlock()

		sessionLogger.Info().
			Str(logging.FieldNewState, string(newState)).
			Int64(logging.FieldCount, session.RelayCount).
			Msg("session lifecycle complete")
	}
}

// HasPendingSessions returns true if there are sessions not yet settled.
func (m *SessionLifecycleManager) HasPendingSessions() bool {
	m.activeSessionsMu.RLock()
	defer m.activeSessionsMu.RUnlock()
	return len(m.activeSessions) > 0
}

// GetPendingSessionCount returns the count of sessions pending settlement.
func (m *SessionLifecycleManager) GetPendingSessionCount() int {
	m.activeSessionsMu.RLock()
	defer m.activeSessionsMu.RUnlock()
	return len(m.activeSessions)
}

// WaitForSettlement waits for all pending sessions to settle.
func (m *SessionLifecycleManager) WaitForSettlement(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !m.HasPendingSessions() {
				return nil
			}

			m.logger.Debug().
				Int("pending", m.GetPendingSessionCount()).
				Msg("waiting for sessions to settle")
		}
	}
}

// Close gracefully shuts down the lifecycle manager.
func (m *SessionLifecycleManager) Close() error {
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

	m.logger.Info().Msg("session lifecycle manager closed")
	return nil
}

// SessionWindow represents the timing windows for a session.
type SessionWindow struct {
	SessionEndHeight int64
	GracePeriodEnd   int64
	ClaimWindowOpen  int64
	ClaimWindowClose int64
	ProofWindowOpen  int64
	ProofWindowClose int64
}

// CalculateSessionWindow calculates all timing windows for a session.
func CalculateSessionWindow(params *sharedtypes.Params, sessionEndHeight int64) SessionWindow {
	return SessionWindow{
		SessionEndHeight: sessionEndHeight,
		GracePeriodEnd:   sessionEndHeight + int64(params.GetGracePeriodEndOffsetBlocks()),
		ClaimWindowOpen:  sharedtypes.GetClaimWindowOpenHeight(params, sessionEndHeight),
		ClaimWindowClose: sharedtypes.GetClaimWindowCloseHeight(params, sessionEndHeight),
		ProofWindowOpen:  sharedtypes.GetProofWindowOpenHeight(params, sessionEndHeight),
		ProofWindowClose: sharedtypes.GetProofWindowCloseHeight(params, sessionEndHeight),
	}
}

// IsInClaimWindow returns true if the current height is within the claim window.
func (w SessionWindow) IsInClaimWindow(currentHeight int64) bool {
	return currentHeight >= w.ClaimWindowOpen && currentHeight < w.ClaimWindowClose
}

// IsInProofWindow returns true if the current height is within the proof window.
func (w SessionWindow) IsInProofWindow(currentHeight int64) bool {
	return currentHeight >= w.ProofWindowOpen && currentHeight < w.ProofWindowClose
}

// BlocksUntilClaimWindowClose returns blocks remaining until claim window closes.
func (w SessionWindow) BlocksUntilClaimWindowClose(currentHeight int64) int64 {
	if currentHeight >= w.ClaimWindowClose {
		return 0
	}
	return w.ClaimWindowClose - currentHeight
}

// BlocksUntilProofWindowClose returns blocks remaining until proof window closes.
func (w SessionWindow) BlocksUntilProofWindowClose(currentHeight int64) int64 {
	if currentHeight >= w.ProofWindowClose {
		return 0
	}
	return w.ProofWindowClose - currentHeight
}
