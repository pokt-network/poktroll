package miner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// SupplierUpdateAction represents the type of supplier update.
type SupplierUpdateAction string

const (
	SupplierUpdateActionAdd      SupplierUpdateAction = "add"
	SupplierUpdateActionUpdate   SupplierUpdateAction = "update"
	SupplierUpdateActionDraining SupplierUpdateAction = "draining"
	SupplierUpdateActionRemove   SupplierUpdateAction = "remove"
)

// SupplierRegistryData is the data stored in Redis for each supplier.
type SupplierRegistryData struct {
	OperatorAddr string   `json:"operator_addr"`
	Services     []string `json:"services"`
	Status       string   `json:"status"` // "active", "draining"
	UpdatedAt    int64    `json:"updated_at"`
}

// SupplierUpdateEvent is published to the Redis channel when suppliers change.
type SupplierUpdateEvent struct {
	Action       SupplierUpdateAction `json:"action"`
	OperatorAddr string               `json:"operator_addr"`
	Services     []string             `json:"services,omitempty"`
	Reason       string               `json:"reason,omitempty"`
}

// SupplierRegistryConfig contains configuration for the SupplierRegistry.
type SupplierRegistryConfig struct {
	// KeyPrefix is the prefix for supplier registry keys.
	// Default: "ha:suppliers"
	KeyPrefix string

	// IndexKey is the key for the supplier index set.
	// Default: "ha:suppliers:index"
	IndexKey string

	// EventChannel is the Redis channel for supplier updates.
	// Default: "ha:events:supplier_update"
	EventChannel string
}

// SupplierRegistry manages supplier registration in Redis.
// It allows relayers to discover available suppliers and their services.
type SupplierRegistry struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	config      SupplierRegistryConfig
}

// NewSupplierRegistry creates a new supplier registry.
func NewSupplierRegistry(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	config SupplierRegistryConfig,
) *SupplierRegistry {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ha:suppliers"
	}
	if config.IndexKey == "" {
		config.IndexKey = "ha:suppliers:index"
	}
	if config.EventChannel == "" {
		config.EventChannel = "ha:events:supplier_update"
	}

	return &SupplierRegistry{
		logger:      logging.ForComponent(logger, logging.ComponentSupplierRegistry),
		redisClient: redisClient,
		config:      config,
	}
}

// PublishSupplierUpdate publishes a supplier update to Redis.
// It updates the supplier data and publishes an event.
func (r *SupplierRegistry) PublishSupplierUpdate(
	ctx context.Context,
	action SupplierUpdateAction,
	operatorAddr string,
	services []string,
) error {
	key := fmt.Sprintf("%s:%s", r.config.KeyPrefix, operatorAddr)

	switch action {
	case SupplierUpdateActionAdd, SupplierUpdateActionUpdate:
		// Set supplier data
		data := SupplierRegistryData{
			OperatorAddr: operatorAddr,
			Services:     services,
			Status:       "active",
			UpdatedAt:    time.Now().Unix(),
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal supplier data: %w", err)
		}

		if err := r.redisClient.Set(ctx, key, jsonData, 0).Err(); err != nil {
			return fmt.Errorf("failed to set supplier data: %w", err)
		}

		// Add to index
		if err := r.redisClient.SAdd(ctx, r.config.IndexKey, operatorAddr).Err(); err != nil {
			return fmt.Errorf("failed to add to supplier index: %w", err)
		}

	case SupplierUpdateActionDraining:
		// Update status to draining
		data := SupplierRegistryData{
			OperatorAddr: operatorAddr,
			Services:     services,
			Status:       "draining",
			UpdatedAt:    time.Now().Unix(),
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal supplier data: %w", err)
		}

		if err := r.redisClient.Set(ctx, key, jsonData, 0).Err(); err != nil {
			return fmt.Errorf("failed to set supplier data: %w", err)
		}

	case SupplierUpdateActionRemove:
		// Remove supplier data
		if err := r.redisClient.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete supplier data: %w", err)
		}

		// Remove from index
		if err := r.redisClient.SRem(ctx, r.config.IndexKey, operatorAddr).Err(); err != nil {
			return fmt.Errorf("failed to remove from supplier index: %w", err)
		}
	}

	// Publish event
	event := SupplierUpdateEvent{
		Action:       action,
		OperatorAddr: operatorAddr,
		Services:     services,
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := r.redisClient.Publish(ctx, r.config.EventChannel, eventData).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	r.logger.Debug().
		Str("action", string(action)).
		Str("operator", operatorAddr).
		Msg("published supplier update")

	supplierRegistryUpdatesTotal.WithLabelValues(string(action)).Inc()

	return nil
}

// GetSupplier retrieves supplier data from Redis.
func (r *SupplierRegistry) GetSupplier(ctx context.Context, operatorAddr string) (*SupplierRegistryData, error) {
	key := fmt.Sprintf("%s:%s", r.config.KeyPrefix, operatorAddr)

	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get supplier data: %w", err)
	}

	var supplierData SupplierRegistryData
	if err := json.Unmarshal(data, &supplierData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier data: %w", err)
	}

	return &supplierData, nil
}

// ListSuppliers returns all registered supplier addresses.
func (r *SupplierRegistry) ListSuppliers(ctx context.Context) ([]string, error) {
	suppliers, err := r.redisClient.SMembers(ctx, r.config.IndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list suppliers: %w", err)
	}

	return suppliers, nil
}

// GetAllSuppliers retrieves data for all registered suppliers.
func (r *SupplierRegistry) GetAllSuppliers(ctx context.Context) (map[string]*SupplierRegistryData, error) {
	suppliers, err := r.ListSuppliers(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*SupplierRegistryData)
	for _, addr := range suppliers {
		data, err := r.GetSupplier(ctx, addr)
		if err != nil {
			r.logger.Warn().
				Err(err).
				Str("operator", addr).
				Msg("failed to get supplier data")
			continue
		}
		if data != nil {
			result[addr] = data
		}
	}

	return result, nil
}

// SubscribeToUpdates subscribes to supplier update events.
// Returns a channel that receives update events.
func (r *SupplierRegistry) SubscribeToUpdates(ctx context.Context) <-chan *SupplierUpdateEvent {
	eventCh := make(chan *SupplierUpdateEvent, 100)

	pubsub := r.redisClient.Subscribe(ctx, r.config.EventChannel)

	go func() {
		defer close(eventCh)
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-pubsub.Channel():
				if !ok {
					return
				}

				var event SupplierUpdateEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					r.logger.Warn().
						Err(err).
						Msg("failed to unmarshal supplier update event")
					continue
				}

				select {
				case eventCh <- &event:
				default:
					r.logger.Warn().Msg("supplier update channel full, dropping event")
				}
			}
		}
	}()

	return eventCh
}

// ClearAll removes all supplier data from Redis.
// Used primarily for testing.
func (r *SupplierRegistry) ClearAll(ctx context.Context) error {
	suppliers, err := r.ListSuppliers(ctx)
	if err != nil {
		return err
	}

	for _, addr := range suppliers {
		key := fmt.Sprintf("%s:%s", r.config.KeyPrefix, addr)
		r.redisClient.Del(ctx, key)
	}

	r.redisClient.Del(ctx, r.config.IndexKey)

	return nil
}
