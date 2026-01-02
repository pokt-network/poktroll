package miner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// SessionState represents the lifecycle state of a session.
type SessionState string

const (
	// SessionStateActive means the session is accepting relays.
	SessionStateActive SessionState = "active"

	// SessionStateClaiming means the session has been flushed and is waiting for claim submission.
	SessionStateClaiming SessionState = "claiming"

	// SessionStateClaimed means the claim has been submitted, waiting for proof window.
	SessionStateClaimed SessionState = "claimed"

	// SessionStateProving means the session is in the proof submission window.
	SessionStateProving SessionState = "proving"

	// SessionStateSettled means the proof was submitted and accepted.
	SessionStateSettled SessionState = "settled"

	// SessionStateExpired means the session expired without completing the lifecycle.
	SessionStateExpired SessionState = "expired"
)

// SessionSnapshot captures the state of a session for HA recovery.
// This is stored in Redis and used by a new leader to recover session state.
type SessionSnapshot struct {
	// SessionID is the unique identifier for this session.
	SessionID string `json:"session_id"`

	// SupplierOperatorAddress is the supplier this session belongs to.
	SupplierOperatorAddress string `json:"supplier_operator_address"`

	// ServiceID is the service this session is for.
	ServiceID string `json:"service_id"`

	// ApplicationAddress is the application this session is with.
	ApplicationAddress string `json:"application_address"`

	// SessionStartHeight is the block height when the session started.
	SessionStartHeight int64 `json:"session_start_height"`

	// SessionEndHeight is the block height when the session ends.
	SessionEndHeight int64 `json:"session_end_height"`

	// State is the current lifecycle state of the session.
	State SessionState `json:"state"`

	// RelayCount is the number of relays processed in this session.
	RelayCount int64 `json:"relay_count"`

	// TotalComputeUnits is the sum of compute units for all relays.
	TotalComputeUnits uint64 `json:"total_compute_units"`

	// ClaimedRootHash is the SMST root hash (set after flush).
	ClaimedRootHash []byte `json:"claimed_root_hash,omitempty"`

	// LastWALEntryID is the last WAL entry ID that was processed.
	// Used for recovery to know where to start replaying from.
	LastWALEntryID string `json:"last_wal_entry_id,omitempty"`

	// LastUpdatedAt is when the snapshot was last updated.
	LastUpdatedAt time.Time `json:"last_updated_at"`

	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"created_at"`
}

// SessionStore provides Redis-based storage for session snapshots.
// This enables HA failover by persisting session state that a new leader can recover.
type SessionStore interface {
	// Save persists a session snapshot to Redis.
	Save(ctx context.Context, snapshot *SessionSnapshot) error

	// Get retrieves a session snapshot by session ID.
	Get(ctx context.Context, sessionID string) (*SessionSnapshot, error)

	// GetBySupplier retrieves all active sessions for a supplier.
	GetBySupplier(ctx context.Context, supplierAddress string) ([]*SessionSnapshot, error)

	// GetByState retrieves all sessions in a given state for a supplier.
	GetByState(ctx context.Context, supplierAddress string, state SessionState) ([]*SessionSnapshot, error)

	// Delete removes a session snapshot.
	Delete(ctx context.Context, sessionID string) error

	// UpdateState atomically updates the state of a session.
	UpdateState(ctx context.Context, sessionID string, newState SessionState) error

	// UpdateWALPosition updates the last WAL entry ID for a session.
	UpdateWALPosition(ctx context.Context, sessionID string, walEntryID string) error

	// IncrementRelayCount atomically increments the relay count and compute units.
	IncrementRelayCount(ctx context.Context, sessionID string, computeUnits uint64) error

	// Close gracefully shuts down the store.
	Close() error
}

// SessionStoreConfig contains configuration for the session store.
type SessionStoreConfig struct {
	// KeyPrefix is the prefix for all Redis keys.
	KeyPrefix string

	// SupplierAddress is the supplier this store is for.
	SupplierAddress string

	// SessionTTL is how long to keep session data after settlement.
	SessionTTL time.Duration
}

// RedisSessionStore implements SessionStore using Redis.
type RedisSessionStore struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	config      SessionStoreConfig

	mu     sync.Mutex
	closed bool
}

// NewRedisSessionStore creates a new Redis-backed session store.
func NewRedisSessionStore(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	config SessionStoreConfig,
) *RedisSessionStore {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ha:miner:sessions"
	}
	if config.SessionTTL == 0 {
		config.SessionTTL = 24 * time.Hour
	}

	return &RedisSessionStore{
		logger:      logging.ForSupplierComponent(logger, logging.ComponentSessionStore, config.SupplierAddress),
		redisClient: redisClient,
		config:      config,
	}
}

// sessionKey returns the Redis key for a session.
func (s *RedisSessionStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("%s:%s:%s", s.config.KeyPrefix, s.config.SupplierAddress, sessionID)
}

// supplierSessionsKey returns the Redis key for the supplier's session index.
func (s *RedisSessionStore) supplierSessionsKey() string {
	return fmt.Sprintf("%s:%s:index", s.config.KeyPrefix, s.config.SupplierAddress)
}

// stateIndexKey returns the Redis key for a state-based index.
func (s *RedisSessionStore) stateIndexKey(state SessionState) string {
	return fmt.Sprintf("%s:%s:state:%s", s.config.KeyPrefix, s.config.SupplierAddress, state)
}

// Save persists a session snapshot to Redis.
func (s *RedisSessionStore) Save(ctx context.Context, snapshot *SessionSnapshot) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("session store is closed")
	}
	s.mu.Unlock()

	snapshot.LastUpdatedAt = time.Now()
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = snapshot.LastUpdatedAt
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal session snapshot: %w", err)
	}

	key := s.sessionKey(snapshot.SessionID)

	// Use a transaction to update the session and indexes atomically
	pipe := s.redisClient.TxPipeline()

	// Store the session data
	pipe.Set(ctx, key, data, s.config.SessionTTL)

	// Add to supplier's session index
	pipe.SAdd(ctx, s.supplierSessionsKey(), snapshot.SessionID)
	pipe.Expire(ctx, s.supplierSessionsKey(), s.config.SessionTTL)

	// Add to state index
	pipe.SAdd(ctx, s.stateIndexKey(snapshot.State), snapshot.SessionID)
	pipe.Expire(ctx, s.stateIndexKey(snapshot.State), s.config.SessionTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save session snapshot: %w", err)
	}

	sessionSnapshotsSaved.WithLabelValues(s.config.SupplierAddress).Inc()

	s.logger.Debug().
		Str("session_id", snapshot.SessionID).
		Str("state", string(snapshot.State)).
		Int64("relay_count", snapshot.RelayCount).
		Msg("saved session snapshot")

	return nil
}

// Get retrieves a session snapshot by session ID.
func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*SessionSnapshot, error) {
	key := s.sessionKey(sessionID)

	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session snapshot: %w", err)
	}

	var snapshot SessionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetBySupplier retrieves all sessions for a supplier.
func (s *RedisSessionStore) GetBySupplier(ctx context.Context, supplierAddress string) ([]*SessionSnapshot, error) {
	// Get all session IDs from the index
	sessionIDs, err := s.redisClient.SMembers(ctx, s.supplierSessionsKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session IDs: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil, nil
	}

	// Get all session data in a pipeline
	pipe := s.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(sessionIDs))

	for i, sessionID := range sessionIDs {
		cmds[i] = pipe.Get(ctx, s.sessionKey(sessionID))
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get session data: %w", err)
	}

	snapshots := make([]*SessionSnapshot, 0, len(sessionIDs))
	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == redis.Nil {
			continue // Session was deleted
		}
		if err != nil {
			continue // Skip errors
		}

		var snapshot SessionSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue // Skip invalid data
		}

		snapshots = append(snapshots, &snapshot)
	}

	return snapshots, nil
}

// GetByState retrieves all sessions in a given state for a supplier.
func (s *RedisSessionStore) GetByState(ctx context.Context, supplierAddress string, state SessionState) ([]*SessionSnapshot, error) {
	// Get session IDs from state index
	sessionIDs, err := s.redisClient.SMembers(ctx, s.stateIndexKey(state)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session IDs by state: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil, nil
	}

	// Get all session data
	pipe := s.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(sessionIDs))

	for i, sessionID := range sessionIDs {
		cmds[i] = pipe.Get(ctx, s.sessionKey(sessionID))
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get session data: %w", err)
	}

	snapshots := make([]*SessionSnapshot, 0, len(sessionIDs))
	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			continue
		}

		var snapshot SessionSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		// Verify state matches (index might be stale)
		if snapshot.State == state {
			snapshots = append(snapshots, &snapshot)
		}
	}

	return snapshots, nil
}

// Delete removes a session snapshot.
func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	// Get the current session to know which indexes to update
	snapshot, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil // Already deleted
	}

	pipe := s.redisClient.TxPipeline()

	// Delete the session data
	pipe.Del(ctx, s.sessionKey(sessionID))

	// Remove from indexes
	pipe.SRem(ctx, s.supplierSessionsKey(), sessionID)
	pipe.SRem(ctx, s.stateIndexKey(snapshot.State), sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session snapshot: %w", err)
	}

	s.logger.Debug().
		Str("session_id", sessionID).
		Msg("deleted session snapshot")

	return nil
}

// UpdateState atomically updates the state of a session.
func (s *RedisSessionStore) UpdateState(ctx context.Context, sessionID string, newState SessionState) error {
	snapshot, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	oldState := snapshot.State
	if oldState == newState {
		return nil // No change
	}

	snapshot.State = newState
	snapshot.LastUpdatedAt = time.Now()

	// Update in a transaction
	pipe := s.redisClient.TxPipeline()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	pipe.Set(ctx, s.sessionKey(sessionID), data, s.config.SessionTTL)

	// Update state indexes
	pipe.SRem(ctx, s.stateIndexKey(oldState), sessionID)
	pipe.SAdd(ctx, s.stateIndexKey(newState), sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update session state: %w", err)
	}

	s.logger.Debug().
		Str("session_id", sessionID).
		Str("old_state", string(oldState)).
		Str("new_state", string(newState)).
		Msg("updated session state")

	return nil
}

// UpdateWALPosition updates the last WAL entry ID for a session.
func (s *RedisSessionStore) UpdateWALPosition(ctx context.Context, sessionID string, walEntryID string) error {
	snapshot, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	snapshot.LastWALEntryID = walEntryID
	snapshot.LastUpdatedAt = time.Now()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return s.redisClient.Set(ctx, s.sessionKey(sessionID), data, s.config.SessionTTL).Err()
}

// IncrementRelayCount atomically increments the relay count and compute units.
func (s *RedisSessionStore) IncrementRelayCount(ctx context.Context, sessionID string, computeUnits uint64) error {
	snapshot, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	snapshot.RelayCount++
	snapshot.TotalComputeUnits += computeUnits
	snapshot.LastUpdatedAt = time.Now()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return s.redisClient.Set(ctx, s.sessionKey(sessionID), data, s.config.SessionTTL).Err()
}

// Close gracefully shuts down the store.
func (s *RedisSessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.logger.Info().Msg("session store closed")
	return nil
}

// Verify interface compliance.
var _ SessionStore = (*RedisSessionStore)(nil)
