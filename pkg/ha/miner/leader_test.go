package miner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func setupLeaderTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, client
}

func TestNewRedisLeaderElector(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config := LeaderElectorConfig{
		SupplierAddress: "pokt1supplier123",
		InstanceID:      "miner-1",
	}

	callbacks := LeaderCallbacks{}

	elector := NewRedisLeaderElector(logger, client, config, callbacks)
	require.NotNil(t, elector)
	require.Equal(t, "pokt1supplier123", elector.config.SupplierAddress)
	require.Equal(t, "miner-1", elector.config.InstanceID)
	require.Equal(t, DefaultLockTTL, elector.config.LockTTL)
	require.Equal(t, DefaultHeartbeatInterval, elector.config.HeartbeatInterval)
}

func TestRedisLeaderElector_SingleInstance_BecomesLeader(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	var elected atomic.Bool

	config := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-1",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	callbacks := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected.Store(true)
			return nil
		},
	}

	elector := NewRedisLeaderElector(logger, client, config, callbacks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := elector.Start(ctx)
	require.NoError(t, err)

	// Wait for leadership to be acquired
	require.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	require.True(t, elected.Load())

	err = elector.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_TwoInstances_OnlyOneLeader(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	var elected1, elected2 atomic.Bool

	config1 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-1",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	config2 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-2",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	callbacks1 := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected1.Store(true)
			return nil
		},
	}

	callbacks2 := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected2.Store(true)
			return nil
		},
	}

	elector1 := NewRedisLeaderElector(logger, client, config1, callbacks1)
	elector2 := NewRedisLeaderElector(logger, client, config2, callbacks2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start both electors
	err := elector1.Start(ctx)
	require.NoError(t, err)

	err = elector2.Start(ctx)
	require.NoError(t, err)

	// Wait for one to become leader
	require.Eventually(t, func() bool {
		return elector1.IsLeader() || elector2.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	// Only one should be leader
	time.Sleep(500 * time.Millisecond) // Give time for both to try
	require.NotEqual(t, elector1.IsLeader(), elector2.IsLeader(), "exactly one should be leader")

	err = elector1.Close()
	require.NoError(t, err)
	err = elector2.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_Failover(t *testing.T) {
	mr, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	var elected1, elected2, lost1 atomic.Bool

	config1 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-1",
		LockTTL:              1 * time.Second, // Short TTL for faster test
		HeartbeatInterval:    300 * time.Millisecond,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	config2 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-2",
		LockTTL:              1 * time.Second,
		HeartbeatInterval:    300 * time.Millisecond,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	callbacks1 := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected1.Store(true)
			return nil
		},
		OnLost: func(ctx context.Context) {
			lost1.Store(true)
		},
	}

	callbacks2 := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected2.Store(true)
			return nil
		},
	}

	elector1 := NewRedisLeaderElector(logger, client, config1, callbacks1)
	elector2 := NewRedisLeaderElector(logger, client, config2, callbacks2)

	ctx := context.Background()

	// Start first elector - it should become leader
	err := elector1.Start(ctx)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return elector1.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	// Start second elector - it should be standby
	err = elector2.Start(ctx)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)
	require.True(t, elector1.IsLeader())
	require.False(t, elector2.IsLeader())

	// Simulate elector1 crash by closing it without releasing lock
	// Then fast-forward time to expire the lock
	elector1.Close()

	// Fast-forward time in miniredis to expire the lock
	mr.FastForward(2 * time.Second)

	// Elector2 should now become leader
	require.Eventually(t, func() bool {
		return elector2.IsLeader()
	}, 3*time.Second, 100*time.Millisecond)

	require.True(t, elected2.Load())

	err = elector2.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_Resign(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	var elected, lost atomic.Bool

	config := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-1",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	callbacks := LeaderCallbacks{
		OnElected: func(ctx context.Context) error {
			elected.Store(true)
			return nil
		},
		OnLost: func(ctx context.Context) {
			lost.Store(true)
		},
	}

	elector := NewRedisLeaderElector(logger, client, config, callbacks)

	ctx := context.Background()

	err := elector.Start(ctx)
	require.NoError(t, err)

	// Wait for leadership
	require.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	require.True(t, elected.Load())

	// Resign
	err = elector.Resign(ctx)
	require.NoError(t, err)

	require.False(t, elector.IsLeader())
	require.True(t, lost.Load())

	// Verify lock is released
	leaderID, err := elector.LeaderID(ctx)
	require.NoError(t, err)
	require.Empty(t, leaderID)

	err = elector.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_LeaderID(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier123",
		InstanceID:           "miner-1",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	elector := NewRedisLeaderElector(logger, client, config, LeaderCallbacks{})

	ctx := context.Background()

	// No leader initially
	leaderID, err := elector.LeaderID(ctx)
	require.NoError(t, err)
	require.Empty(t, leaderID)

	// Start elector
	err = elector.Start(ctx)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	// Should show our instance ID as leader
	leaderID, err = elector.LeaderID(ctx)
	require.NoError(t, err)
	require.Equal(t, "miner-1", leaderID)

	err = elector.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_Close_Safe(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config := LeaderElectorConfig{
		SupplierAddress: "pokt1supplier123",
		InstanceID:      "miner-1",
	}

	elector := NewRedisLeaderElector(logger, client, config, LeaderCallbacks{})

	// Double close should be safe
	err := elector.Close()
	require.NoError(t, err)

	err = elector.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_Start_AlreadyClosed(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config := LeaderElectorConfig{
		SupplierAddress: "pokt1supplier123",
		InstanceID:      "miner-1",
	}

	elector := NewRedisLeaderElector(logger, client, config, LeaderCallbacks{})
	elector.Close()

	err := elector.Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestRedisLeaderElector_Start_AlreadyStarted(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config := LeaderElectorConfig{
		SupplierAddress: "pokt1supplier123",
		InstanceID:      "miner-1",
	}

	elector := NewRedisLeaderElector(logger, client, config, LeaderCallbacks{})

	ctx := context.Background()
	err := elector.Start(ctx)
	require.NoError(t, err)

	err = elector.Start(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already started")

	err = elector.Close()
	require.NoError(t, err)
}

func TestRedisLeaderElector_DifferentSuppliers_BothLeaders(t *testing.T) {
	_, client := setupLeaderTestRedis(t)
	logger := polyzero.NewLogger()

	config1 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier_A",
		InstanceID:           "miner-1",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	config2 := LeaderElectorConfig{
		SupplierAddress:      "pokt1supplier_B", // Different supplier
		InstanceID:           "miner-2",
		LockTTL:              5 * time.Second,
		AcquireRetryInterval: 100 * time.Millisecond,
	}

	elector1 := NewRedisLeaderElector(logger, client, config1, LeaderCallbacks{})
	elector2 := NewRedisLeaderElector(logger, client, config2, LeaderCallbacks{})

	ctx := context.Background()

	err := elector1.Start(ctx)
	require.NoError(t, err)

	err = elector2.Start(ctx)
	require.NoError(t, err)

	// Both should become leaders (different suppliers)
	require.Eventually(t, func() bool {
		return elector1.IsLeader() && elector2.IsLeader()
	}, 2*time.Second, 50*time.Millisecond)

	err = elector1.Close()
	require.NoError(t, err)
	err = elector2.Close()
	require.NoError(t, err)
}
