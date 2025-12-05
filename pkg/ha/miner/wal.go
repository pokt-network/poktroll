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

const (
	// DefaultWALMaxLen is the default maximum length of a WAL stream.
	// After this, old entries are trimmed.
	DefaultWALMaxLen = 100000

	// DefaultWALRetention is how long to keep WAL entries after checkpoint.
	DefaultWALRetention = 24 * time.Hour
)

// WAL (Write-Ahead Log) provides durable storage for relay entries.
// Relays are written to WAL before updating the SMST, ensuring
// no data loss on crashes.
type WAL interface {
	// Append adds a relay entry to the WAL.
	// Returns the WAL entry ID.
	Append(ctx context.Context, sessionID string, entry *WALEntry) (string, error)

	// AppendBatch adds multiple relay entries to the WAL.
	// Returns the WAL entry IDs.
	AppendBatch(ctx context.Context, sessionID string, entries []*WALEntry) ([]string, error)

	// ReadFrom reads all entries from a given position (exclusive).
	// Pass "0" to read from the beginning.
	ReadFrom(ctx context.Context, sessionID string, fromID string) ([]*WALEntry, error)

	// Checkpoint marks a position as safely persisted.
	// Entries before this position may be trimmed.
	Checkpoint(ctx context.Context, sessionID string, lastID string) error

	// GetCheckpoint returns the last checkpoint position.
	GetCheckpoint(ctx context.Context, sessionID string) (string, error)

	// Trim removes entries older than the checkpoint.
	Trim(ctx context.Context, sessionID string) error

	// Size returns the number of entries in the WAL.
	Size(ctx context.Context, sessionID string) (int64, error)

	// Close gracefully shuts down the WAL.
	Close() error
}

// WALEntry represents a single entry in the WAL.
type WALEntry struct {
	// ID is the WAL entry ID (set after append).
	ID string `json:"id,omitempty"`

	// RelayHash is the hash of the relay.
	RelayHash []byte `json:"relay_hash"`

	// RelayBytes is the serialized relay.
	RelayBytes []byte `json:"relay_bytes"`

	// SessionID is the session this relay belongs to.
	SessionID string `json:"session_id"`

	// SupplierAddress is the supplier operator address.
	SupplierAddress string `json:"supplier_address"`

	// ServiceID is the service ID.
	ServiceID string `json:"service_id"`

	// Timestamp is when the entry was created.
	Timestamp time.Time `json:"timestamp"`

	// ComputeUnits is the compute units for this relay.
	ComputeUnits uint64 `json:"compute_units"`
}

// WALConfig contains configuration for the WAL.
type WALConfig struct {
	// KeyPrefix is the prefix for WAL Redis keys.
	KeyPrefix string

	// SupplierAddress is the supplier this WAL is for.
	SupplierAddress string

	// MaxLen is the maximum number of entries per session.
	MaxLen int64

	// TrimInterval is how often to trim old entries.
	TrimInterval time.Duration
}

// RedisWAL implements WAL using Redis Streams.
type RedisWAL struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	config      WALConfig

	// Lifecycle
	mu       sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewRedisWAL creates a new Redis-backed WAL.
func NewRedisWAL(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	config WALConfig,
) *RedisWAL {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ha:miner:wal"
	}
	if config.MaxLen == 0 {
		config.MaxLen = DefaultWALMaxLen
	}
	if config.TrimInterval == 0 {
		config.TrimInterval = 5 * time.Minute
	}

	return &RedisWAL{
		logger:      logging.ForSupplierComponent(logger, logging.ComponentWAL, config.SupplierAddress),
		redisClient: redisClient,
		config:      config,
	}
}

// Start begins background processes (like periodic trimming).
func (w *RedisWAL) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("WAL is closed")
	}
	ctx, w.cancelFn = context.WithCancel(ctx)
	w.mu.Unlock()

	// Start periodic trim loop
	w.wg.Add(1)
	go w.trimLoop(ctx)

	w.logger.Info().Msg("WAL started")
	return nil
}

// trimLoop periodically trims old entries from all sessions.
func (w *RedisWAL) trimLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.TrimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Trim is called per-session, so we don't do global trim here
			// Individual sessions should call Trim after checkpointing
		}
	}
}

// streamKey returns the Redis stream key for a session's WAL.
func (w *RedisWAL) streamKey(sessionID string) string {
	return fmt.Sprintf("%s:%s:%s", w.config.KeyPrefix, w.config.SupplierAddress, sessionID)
}

// checkpointKey returns the Redis key for a session's checkpoint.
func (w *RedisWAL) checkpointKey(sessionID string) string {
	return fmt.Sprintf("%s:%s:%s:checkpoint", w.config.KeyPrefix, w.config.SupplierAddress, sessionID)
}

// Append adds a relay entry to the WAL.
func (w *RedisWAL) Append(ctx context.Context, sessionID string, entry *WALEntry) (string, error) {
	entry.SessionID = sessionID
	entry.SupplierAddress = w.config.SupplierAddress
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal WAL entry: %w", err)
	}

	// Add to stream with MAXLEN to prevent unbounded growth
	key := w.streamKey(sessionID)
	id, err := w.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: w.config.MaxLen,
		Approx: true, // Use ~ for better performance
		Values: map[string]interface{}{
			"data": data,
		},
	}).Result()

	if err != nil {
		return "", fmt.Errorf("failed to append to WAL: %w", err)
	}

	entry.ID = id
	walAppends.WithLabelValues(w.config.SupplierAddress, sessionID).Inc()

	return id, nil
}

// AppendBatch adds multiple relay entries to the WAL.
func (w *RedisWAL) AppendBatch(ctx context.Context, sessionID string, entries []*WALEntry) ([]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	key := w.streamKey(sessionID)
	ids := make([]string, len(entries))

	// Use pipeline for batch append
	pipe := w.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(entries))

	for i, entry := range entries {
		entry.SessionID = sessionID
		entry.SupplierAddress = w.config.SupplierAddress
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		data, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal WAL entry %d: %w", i, err)
		}

		cmds[i] = pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: key,
			MaxLen: w.config.MaxLen,
			Approx: true,
			Values: map[string]interface{}{
				"data": data,
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to append batch to WAL: %w", err)
	}

	for i, cmd := range cmds {
		id, err := cmd.Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get WAL entry ID %d: %w", i, err)
		}
		ids[i] = id
		entries[i].ID = id
	}

	walAppends.WithLabelValues(w.config.SupplierAddress, sessionID).Add(float64(len(entries)))

	return ids, nil
}

// ReadFrom reads all entries from a given position (exclusive).
// Pass "0" or "" to read from the beginning.
func (w *RedisWAL) ReadFrom(ctx context.Context, sessionID string, fromID string) ([]*WALEntry, error) {
	key := w.streamKey(sessionID)

	// Determine start ID for XRANGE
	startID := "-"
	excludeFirst := false
	if fromID != "" && fromID != "0" && fromID != "-" {
		// Start from the given ID but we'll skip it (exclusive)
		startID = fromID
		excludeFirst = true
	}

	// Use XRANGE for non-blocking read
	messages, err := w.redisClient.XRange(ctx, key, startID, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL: %w", err)
	}

	if len(messages) == 0 {
		return nil, nil
	}

	entries := make([]*WALEntry, 0, len(messages))
	for i, msg := range messages {
		// Skip the first message if we're doing exclusive read and it matches
		if excludeFirst && i == 0 && msg.ID == fromID {
			continue
		}

		data, ok := msg.Values["data"].(string)
		if !ok {
			w.logger.Warn().Str("id", msg.ID).Msg("invalid WAL entry format")
			continue
		}

		var entry WALEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			w.logger.Warn().Err(err).Str("id", msg.ID).Msg("failed to unmarshal WAL entry")
			continue
		}

		entry.ID = msg.ID
		entries = append(entries, &entry)
		walReplays.WithLabelValues(w.config.SupplierAddress, sessionID).Inc()
	}

	return entries, nil
}

// Checkpoint marks a position as safely persisted.
func (w *RedisWAL) Checkpoint(ctx context.Context, sessionID string, lastID string) error {
	key := w.checkpointKey(sessionID)

	err := w.redisClient.Set(ctx, key, lastID, DefaultWALRetention).Err()
	if err != nil {
		return fmt.Errorf("failed to set checkpoint: %w", err)
	}

	walCheckpoints.WithLabelValues(w.config.SupplierAddress, sessionID).Inc()

	w.logger.Debug().
		Str("session_id", sessionID).
		Str("checkpoint_id", lastID).
		Msg("WAL checkpoint created")

	return nil
}

// GetCheckpoint returns the last checkpoint position.
func (w *RedisWAL) GetCheckpoint(ctx context.Context, sessionID string) (string, error) {
	key := w.checkpointKey(sessionID)

	checkpoint, err := w.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "0", nil // No checkpoint
	}
	if err != nil {
		return "", fmt.Errorf("failed to get checkpoint: %w", err)
	}

	return checkpoint, nil
}

// Trim removes entries older than the checkpoint.
func (w *RedisWAL) Trim(ctx context.Context, sessionID string) error {
	checkpoint, err := w.GetCheckpoint(ctx, sessionID)
	if err != nil {
		return err
	}

	if checkpoint == "" || checkpoint == "0" {
		return nil // Nothing to trim
	}

	key := w.streamKey(sessionID)

	// XTRIM MINID to remove entries before checkpoint
	// We use the checkpoint ID as the minimum ID to keep
	trimmed, err := w.redisClient.XTrimMinID(ctx, key, checkpoint).Result()
	if err != nil {
		return fmt.Errorf("failed to trim WAL: %w", err)
	}

	if trimmed > 0 {
		w.logger.Debug().
			Str("session_id", sessionID).
			Int64("trimmed", trimmed).
			Msg("WAL trimmed")
	}

	return nil
}

// Size returns the number of entries in the WAL.
func (w *RedisWAL) Size(ctx context.Context, sessionID string) (int64, error) {
	key := w.streamKey(sessionID)

	size, err := w.redisClient.XLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get WAL size: %w", err)
	}

	walSize.WithLabelValues(w.config.SupplierAddress, sessionID).Set(float64(size))
	return size, nil
}

// DeleteSession removes all WAL data for a session.
func (w *RedisWAL) DeleteSession(ctx context.Context, sessionID string) error {
	streamKey := w.streamKey(sessionID)
	checkpointKey := w.checkpointKey(sessionID)

	pipe := w.redisClient.Pipeline()
	pipe.Del(ctx, streamKey)
	pipe.Del(ctx, checkpointKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session WAL: %w", err)
	}

	w.logger.Debug().
		Str("session_id", sessionID).
		Msg("WAL session deleted")

	return nil
}

// Close gracefully shuts down the WAL.
func (w *RedisWAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	if w.cancelFn != nil {
		w.cancelFn()
	}

	w.wg.Wait()

	w.logger.Info().Msg("WAL closed")
	return nil
}

// Verify interface compliance.
var _ WAL = (*RedisWAL)(nil)
