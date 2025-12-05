package miner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// DefaultLockTTL is how long the lock is held before expiring.
	DefaultLockTTL = 30 * time.Second

	// DefaultHeartbeatInterval is how often to refresh the lock.
	DefaultHeartbeatInterval = 10 * time.Second

	// DefaultAcquireRetryInterval is how often standbys try to acquire leadership.
	DefaultAcquireRetryInterval = 5 * time.Second
)

// LeaderElector manages leader election for a single supplier's miner.
// Only one miner instance can be the leader at a time.
// The leader is responsible for processing relays and updating the SMST.
type LeaderElector interface {
	// Start begins the leader election process.
	// If leadership is acquired, onElected is called.
	// If leadership is lost, onLost is called.
	Start(ctx context.Context) error

	// IsLeader returns true if this instance is currently the leader.
	IsLeader() bool

	// LeaderID returns the ID of the current leader (may be another instance).
	LeaderID(ctx context.Context) (string, error)

	// Resign voluntarily gives up leadership.
	Resign(ctx context.Context) error

	// Close stops the leader election process.
	Close() error
}

// LeaderElectorConfig contains configuration for leader election.
type LeaderElectorConfig struct {
	// SupplierAddress is the supplier this miner is responsible for.
	SupplierAddress string

	// InstanceID is the unique identifier for this miner instance.
	InstanceID string

	// LockTTL is how long the lock is held before expiring.
	LockTTL time.Duration

	// HeartbeatInterval is how often to refresh the lock while leader.
	HeartbeatInterval time.Duration

	// AcquireRetryInterval is how often to try acquiring leadership while standby.
	AcquireRetryInterval time.Duration

	// KeyPrefix is the prefix for Redis keys.
	KeyPrefix string
}

// LeaderCallbacks contains callbacks for leadership changes.
type LeaderCallbacks struct {
	// OnElected is called when this instance becomes the leader.
	OnElected func(ctx context.Context) error

	// OnLost is called when this instance loses leadership.
	OnLost func(ctx context.Context)
}

// RedisLeaderElector implements LeaderElector using Redis.
type RedisLeaderElector struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	config      LeaderElectorConfig
	callbacks   LeaderCallbacks

	// Current leadership state
	isLeader atomic.Bool

	// Lifecycle management
	mu       sync.Mutex
	started  bool
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewRedisLeaderElector creates a new Redis-based leader elector.
func NewRedisLeaderElector(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	config LeaderElectorConfig,
	callbacks LeaderCallbacks,
) *RedisLeaderElector {
	// Apply defaults
	if config.LockTTL == 0 {
		config.LockTTL = DefaultLockTTL
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if config.AcquireRetryInterval == 0 {
		config.AcquireRetryInterval = DefaultAcquireRetryInterval
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ha:miner:leader"
	}

	return &RedisLeaderElector{
		logger:      logger.With(logging.FieldComponent, logging.ComponentLeaderElector, logging.FieldSupplier, config.SupplierAddress, logging.FieldInstance, config.InstanceID),
		redisClient: redisClient,
		config:      config,
		callbacks:   callbacks,
	}
}

// Start begins the leader election process.
func (e *RedisLeaderElector) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return fmt.Errorf("leader elector is closed")
	}
	if e.started {
		e.mu.Unlock()
		return fmt.Errorf("leader elector already started")
	}
	e.started = true
	ctx, e.cancelFn = context.WithCancel(ctx)
	e.mu.Unlock()

	// Start the election loop
	e.wg.Add(1)
	go e.electionLoop(ctx)

	e.logger.Info().Msg("leader elector started")
	return nil
}

// electionLoop continuously tries to acquire/maintain leadership.
func (e *RedisLeaderElector) electionLoop(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// If we're the leader, release the lock
			if e.isLeader.Load() {
				if err := e.releaseLock(context.Background()); err != nil {
					e.logger.Error().Err(err).Msg("failed to release lock on shutdown")
				}
			}
			return
		default:
		}

		if e.isLeader.Load() {
			// We're the leader - maintain heartbeat
			if err := e.heartbeat(ctx); err != nil {
				e.logger.Error().Err(err).Msg("failed to maintain leadership")
				e.loseLeadership(ctx)
			}
			time.Sleep(e.config.HeartbeatInterval)
		} else {
			// We're a standby - try to acquire leadership
			acquired, err := e.tryAcquire(ctx)
			if err != nil {
				e.logger.Error().Err(err).Msg("failed to acquire leadership")
			} else if acquired {
				e.becomeLeader(ctx)
			}
			time.Sleep(e.config.AcquireRetryInterval)
		}
	}
}

// lockKey returns the Redis key for the leader lock.
func (e *RedisLeaderElector) lockKey() string {
	return fmt.Sprintf("%s:%s", e.config.KeyPrefix, e.config.SupplierAddress)
}

// tryAcquire attempts to acquire the leader lock.
func (e *RedisLeaderElector) tryAcquire(ctx context.Context) (bool, error) {
	key := e.lockKey()

	// Try to set the lock with NX (only if not exists)
	ok, err := e.redisClient.SetNX(ctx, key, e.config.InstanceID, e.config.LockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if ok {
		leaderAcquisitions.WithLabelValues(e.config.SupplierAddress).Inc()
		e.logger.Info().Msg("acquired leadership")
	}

	return ok, nil
}

// heartbeat refreshes the leader lock.
func (e *RedisLeaderElector) heartbeat(ctx context.Context) error {
	key := e.lockKey()

	// Use a Lua script to atomically check owner and refresh TTL
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, e.redisClient, []string{key}, e.config.InstanceID, e.config.LockTTL.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result == 0 {
		// We lost the lock (someone else has it or it expired)
		return fmt.Errorf("lost leadership: lock not owned by us")
	}

	leaderHeartbeats.WithLabelValues(e.config.SupplierAddress).Inc()
	return nil
}

// releaseLock releases the leader lock.
func (e *RedisLeaderElector) releaseLock(ctx context.Context) error {
	key := e.lockKey()

	// Use a Lua script to atomically check owner and delete
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	_, err := script.Run(ctx, e.redisClient, []string{key}, e.config.InstanceID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	e.logger.Info().Msg("released leadership lock")
	return nil
}

// becomeLeader handles the transition to leader state.
func (e *RedisLeaderElector) becomeLeader(ctx context.Context) {
	e.isLeader.Store(true)
	leaderStatus.WithLabelValues(e.config.SupplierAddress, e.config.InstanceID).Set(1)

	e.logger.Info().Msg("became leader - starting miner operations")

	if e.callbacks.OnElected != nil {
		if err := e.callbacks.OnElected(ctx); err != nil {
			e.logger.Error().Err(err).Msg("onElected callback failed")
			// If callback fails, resign leadership
			e.loseLeadership(ctx)
			if releaseErr := e.releaseLock(ctx); releaseErr != nil {
				e.logger.Error().Err(releaseErr).Msg("failed to release lock after callback failure")
			}
		}
	}
}

// loseLeadership handles the transition from leader to standby.
func (e *RedisLeaderElector) loseLeadership(ctx context.Context) {
	wasLeader := e.isLeader.Swap(false)
	if !wasLeader {
		return
	}

	leaderStatus.WithLabelValues(e.config.SupplierAddress, e.config.InstanceID).Set(0)
	leaderLosses.WithLabelValues(e.config.SupplierAddress).Inc()

	e.logger.Warn().Msg("lost leadership - stopping miner operations")

	if e.callbacks.OnLost != nil {
		e.callbacks.OnLost(ctx)
	}
}

// IsLeader returns true if this instance is currently the leader.
func (e *RedisLeaderElector) IsLeader() bool {
	return e.isLeader.Load()
}

// LeaderID returns the ID of the current leader.
func (e *RedisLeaderElector) LeaderID(ctx context.Context) (string, error) {
	key := e.lockKey()
	id, err := e.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // No leader
	}
	if err != nil {
		return "", fmt.Errorf("failed to get leader ID: %w", err)
	}
	return id, nil
}

// Resign voluntarily gives up leadership.
func (e *RedisLeaderElector) Resign(ctx context.Context) error {
	if !e.isLeader.Load() {
		return nil
	}

	e.loseLeadership(ctx)
	return e.releaseLock(ctx)
}

// Close stops the leader election process.
func (e *RedisLeaderElector) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}
	e.closed = true

	if e.cancelFn != nil {
		e.cancelFn()
	}

	e.wg.Wait()

	e.logger.Info().Msg("leader elector closed")
	return nil
}

// Verify interface compliance.
var _ LeaderElector = (*RedisLeaderElector)(nil)
