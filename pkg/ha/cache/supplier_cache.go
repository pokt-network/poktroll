package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// DefaultSupplierKeyPrefix is the default Redis key prefix for supplier state.
	DefaultSupplierKeyPrefix = "ha:supplier"

	// SupplierStatusActive indicates the supplier is active and accepting relays.
	SupplierStatusActive = "active"

	// SupplierStatusUnstaking indicates the supplier is unstaking.
	// Check UnstakeSessionEndHeight to determine if still in grace period.
	SupplierStatusUnstaking = "unstaking"
)

// SupplierState represents the cached state of a supplier.
// This is stored in Redis and shared between miner (writer) and relayer (reader).
type SupplierState struct {
	// Status is the supplier's current status: "active" or "unstaking"
	Status string `json:"status"`

	// Services is the list of service IDs this supplier is staked for.
	Services []string `json:"services"`

	// UnstakeSessionEndHeight is the session end height when unstaking takes effect.
	// 0 means the supplier is active (not unstaking).
	// >0 means the supplier is unstaking and will be fully unstaked at this session end height.
	//
	// TODO_IMPROVE: Handle unstaking grace period properly.
	// When supplier unstakes, they should continue serving until current session ends.
	// The unstaking takes effect at the next session boundary to prevent
	// breaking sessions halfway for both gateway and supplier.
	// This requires comparing against current session end height.
	UnstakeSessionEndHeight uint64 `json:"unstake_session_end_height"`

	// OperatorAddress is the supplier's operator address.
	OperatorAddress string `json:"operator_address"`

	// OwnerAddress is the supplier's owner address.
	OwnerAddress string `json:"owner_address"`

	// LastUpdated is the Unix timestamp when this state was last updated.
	LastUpdated int64 `json:"last_updated"`

	// UpdatedBy identifies which miner instance updated this state (for debugging).
	UpdatedBy string `json:"updated_by,omitempty"`
}

// IsActive returns true if the supplier is active and should accept relays.
// TODO_IMPROVE: Add session height comparison for proper unstaking grace period.
// For now, any unstaking supplier is considered inactive.
func (s *SupplierState) IsActive() bool {
	return s.Status == SupplierStatusActive && s.UnstakeSessionEndHeight == 0
}

// IsActiveForService returns true if the supplier is active for the given service.
func (s *SupplierState) IsActiveForService(serviceID string) bool {
	if !s.IsActive() {
		return false
	}
	for _, svc := range s.Services {
		if svc == serviceID {
			return true
		}
	}
	return false
}

// SupplierCache provides read/write access to the shared supplier state cache in Redis.
type SupplierCache struct {
	logger    polylog.Logger
	redis     *redis.Client
	keyPrefix string
	failOpen  bool
}

// SupplierCacheConfig contains configuration for SupplierCache.
type SupplierCacheConfig struct {
	// KeyPrefix is the Redis key prefix for supplier state.
	KeyPrefix string

	// FailOpen determines behavior when Redis is unavailable.
	// If true, treat supplier as active when cache unavailable (safer for traffic).
	// If false, treat supplier as inactive when cache unavailable (safer for validation).
	FailOpen bool
}

// NewSupplierCache creates a new SupplierCache.
func NewSupplierCache(
	logger polylog.Logger,
	redisClient *redis.Client,
	config SupplierCacheConfig,
) *SupplierCache {
	keyPrefix := config.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = DefaultSupplierKeyPrefix
	}

	return &SupplierCache{
		logger:    logger,
		redis:     redisClient,
		keyPrefix: keyPrefix,
		failOpen:  config.FailOpen,
	}
}

// supplierKey returns the Redis key for a supplier's state.
func (c *SupplierCache) supplierKey(operatorAddress string) string {
	return fmt.Sprintf("%s:%s", c.keyPrefix, operatorAddress)
}

// GetSupplierState retrieves a supplier's state from the cache.
// Returns nil if the supplier is not in the cache.
// If Redis is unavailable and FailOpen is true, returns a synthetic "active" state.
func (c *SupplierCache) GetSupplierState(ctx context.Context, operatorAddress string) (*SupplierState, error) {
	key := c.supplierKey(operatorAddress)

	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Supplier not in cache
			return nil, nil
		}

		// Redis error
		c.logger.Warn().
			Err(err).
			Str("operator_address", operatorAddress).
			Bool("fail_open", c.failOpen).
			Msg("failed to get supplier state from cache")

		if c.failOpen {
			// Return synthetic active state to avoid blocking traffic
			c.logger.Warn().
				Str("operator_address", operatorAddress).
				Msg("fail-open: treating supplier as active due to cache error")
			return &SupplierState{
				Status:          SupplierStatusActive,
				OperatorAddress: operatorAddress,
			}, nil
		}

		return nil, fmt.Errorf("failed to get supplier state: %w", err)
	}

	var state SupplierState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier state: %w", err)
	}

	return &state, nil
}

// SetSupplierState stores a supplier's state in the cache.
// This is typically called by the miner to update supplier state.
func (c *SupplierCache) SetSupplierState(ctx context.Context, state *SupplierState) error {
	if state.OperatorAddress == "" {
		return fmt.Errorf("operator_address is required")
	}

	// Update timestamp
	state.LastUpdated = time.Now().Unix()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal supplier state: %w", err)
	}

	key := c.supplierKey(state.OperatorAddress)

	// No TTL - explicit state management only
	if err := c.redis.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to set supplier state: %w", err)
	}

	c.logger.Debug().
		Str("operator_address", state.OperatorAddress).
		Str("status", state.Status).
		Uint64("unstake_session_end_height", state.UnstakeSessionEndHeight).
		Msg("updated supplier state in cache")

	return nil
}

// DeleteSupplierState removes a supplier's state from the cache.
// This should be called when a supplier is fully unstaked.
func (c *SupplierCache) DeleteSupplierState(ctx context.Context, operatorAddress string) error {
	key := c.supplierKey(operatorAddress)

	if err := c.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete supplier state: %w", err)
	}

	c.logger.Info().
		Str("operator_address", operatorAddress).
		Msg("deleted supplier state from cache")

	return nil
}

// IsSupplierActiveForService checks if a supplier is active for a service.
// This is a convenience method that combines GetSupplierState and IsActiveForService.
// Returns (true, nil) if supplier is active for the service.
// Returns (false, nil) if supplier is not active or not in cache.
// Returns (false, error) if there was a cache error and FailOpen is false.
func (c *SupplierCache) IsSupplierActiveForService(
	ctx context.Context,
	operatorAddress string,
	serviceID string,
) (bool, error) {
	state, err := c.GetSupplierState(ctx, operatorAddress)
	if err != nil {
		return false, err
	}

	if state == nil {
		// Supplier not in cache
		if c.failOpen {
			c.logger.Warn().
				Str("operator_address", operatorAddress).
				Str("service_id", serviceID).
				Msg("fail-open: supplier not in cache, treating as active")
			return true, nil
		}
		return false, nil
	}

	return state.IsActiveForService(serviceID), nil
}

// GetAllSupplierStates returns all supplier states from the cache.
// This is useful for debugging and monitoring.
func (c *SupplierCache) GetAllSupplierStates(ctx context.Context) (map[string]*SupplierState, error) {
	pattern := fmt.Sprintf("%s:*", c.keyPrefix)
	keys, err := c.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier keys: %w", err)
	}

	states := make(map[string]*SupplierState)
	for _, key := range keys {
		data, err := c.redis.Get(ctx, key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, fmt.Errorf("failed to get supplier state for key %s: %w", key, err)
		}

		var state SupplierState
		if err := json.Unmarshal(data, &state); err != nil {
			c.logger.Warn().
				Err(err).
				Str("key", key).
				Msg("failed to unmarshal supplier state, skipping")
			continue
		}

		states[state.OperatorAddress] = &state
	}

	return states, nil
}
