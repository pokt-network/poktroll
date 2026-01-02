//go:build test

package miner

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func newTestRedisSessionStore(t *testing.T, supplierAddress string) (*RedisSessionStore, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := polyzero.NewLogger()

	store := NewRedisSessionStore(logger, client, SessionStoreConfig{
		KeyPrefix:       "test:sessions",
		SupplierAddress: supplierAddress,
		SessionTTL:      time.Hour,
	})

	t.Cleanup(func() {
		store.Close()
		client.Close()
		mr.Close()
	})

	return store, mr
}

func TestRedisSessionStore_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-123",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		ApplicationAddress:      "app1",
		SessionStartHeight:      100,
		SessionEndHeight:        110,
		State:                   SessionStateActive,
		RelayCount:              50,
		TotalComputeUnits:       5000,
	}

	// Save
	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Verify timestamps are set
	require.False(t, snapshot.CreatedAt.IsZero())
	require.False(t, snapshot.LastUpdatedAt.IsZero())

	// Get
	retrieved, err := store.Get(ctx, "session-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	require.Equal(t, snapshot.SessionID, retrieved.SessionID)
	require.Equal(t, snapshot.SupplierOperatorAddress, retrieved.SupplierOperatorAddress)
	require.Equal(t, snapshot.ServiceID, retrieved.ServiceID)
	require.Equal(t, snapshot.ApplicationAddress, retrieved.ApplicationAddress)
	require.Equal(t, snapshot.State, retrieved.State)
	require.Equal(t, snapshot.RelayCount, retrieved.RelayCount)
	require.Equal(t, snapshot.TotalComputeUnits, retrieved.TotalComputeUnits)
}

func TestRedisSessionStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Get non-existent session
	retrieved, err := store.Get(ctx, "nonexistent")
	require.NoError(t, err)
	require.Nil(t, retrieved)
}

func TestRedisSessionStore_GetBySupplier(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		snapshot := &SessionSnapshot{
			SessionID:               "session-" + string(rune('a'+i)),
			SupplierOperatorAddress: "supplier1",
			ServiceID:               "svc1",
			State:                   SessionStateActive,
		}
		err := store.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get all sessions for supplier
	sessions, err := store.GetBySupplier(ctx, "supplier1")
	require.NoError(t, err)
	require.Len(t, sessions, 5)
}

func TestRedisSessionStore_GetByState(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Create sessions in different states
	states := []SessionState{
		SessionStateActive,
		SessionStateActive,
		SessionStateClaiming,
		SessionStateClaimed,
		SessionStateProving,
	}

	for i, state := range states {
		snapshot := &SessionSnapshot{
			SessionID:               "session-" + string(rune('a'+i)),
			SupplierOperatorAddress: "supplier1",
			ServiceID:               "svc1",
			State:                   state,
		}
		err := store.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get only active sessions
	activeSessions, err := store.GetByState(ctx, "supplier1", SessionStateActive)
	require.NoError(t, err)
	require.Len(t, activeSessions, 2)

	// Get claiming sessions
	claimingSessions, err := store.GetByState(ctx, "supplier1", SessionStateClaiming)
	require.NoError(t, err)
	require.Len(t, claimingSessions, 1)

	// Get settled sessions (none exist)
	settledSessions, err := store.GetByState(ctx, "supplier1", SessionStateSettled)
	require.NoError(t, err)
	require.Len(t, settledSessions, 0)
}

func TestRedisSessionStore_Delete(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-to-delete",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}

	// Save
	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := store.Get(ctx, "session-to-delete")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete
	err = store.Delete(ctx, "session-to-delete")
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err = store.Get(ctx, "session-to-delete")
	require.NoError(t, err)
	require.Nil(t, retrieved)

	// Verify removed from indexes
	sessions, err := store.GetBySupplier(ctx, "supplier1")
	require.NoError(t, err)
	require.Len(t, sessions, 0)
}

func TestRedisSessionStore_DeleteNonExistent(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Delete non-existent should not error
	err := store.Delete(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestRedisSessionStore_UpdateState(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-state-test",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}

	// Save
	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Update state
	err = store.UpdateState(ctx, "session-state-test", SessionStateClaiming)
	require.NoError(t, err)

	// Verify state changed
	retrieved, err := store.Get(ctx, "session-state-test")
	require.NoError(t, err)
	require.Equal(t, SessionStateClaiming, retrieved.State)

	// Verify index updated - should no longer be in active index
	activeSessions, err := store.GetByState(ctx, "supplier1", SessionStateActive)
	require.NoError(t, err)
	require.Len(t, activeSessions, 0)

	// Should be in claiming index
	claimingSessions, err := store.GetByState(ctx, "supplier1", SessionStateClaiming)
	require.NoError(t, err)
	require.Len(t, claimingSessions, 1)
}

func TestRedisSessionStore_UpdateStateNoChange(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-no-change",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}

	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	originalUpdatedAt := snapshot.LastUpdatedAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update to same state
	err = store.UpdateState(ctx, "session-no-change", SessionStateActive)
	require.NoError(t, err)

	// Verify LastUpdatedAt didn't change (no-op)
	retrieved, err := store.Get(ctx, "session-no-change")
	require.NoError(t, err)
	require.Equal(t, originalUpdatedAt.Unix(), retrieved.LastUpdatedAt.Unix())
}

func TestRedisSessionStore_UpdateStateNotFound(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	err := store.UpdateState(ctx, "nonexistent", SessionStateClaiming)
	require.Error(t, err)
	require.Contains(t, err.Error(), "session not found")
}

func TestRedisSessionStore_UpdateWALPosition(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-wal",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}

	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Update WAL position
	err = store.UpdateWALPosition(ctx, "session-wal", "1234567890-0")
	require.NoError(t, err)

	// Verify
	retrieved, err := store.Get(ctx, "session-wal")
	require.NoError(t, err)
	require.Equal(t, "1234567890-0", retrieved.LastWALEntryID)
}

func TestRedisSessionStore_IncrementRelayCount(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	snapshot := &SessionSnapshot{
		SessionID:               "session-relay",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
		RelayCount:              0,
		TotalComputeUnits:       0,
	}

	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Increment multiple times
	for i := 0; i < 10; i++ {
		err = store.IncrementRelayCount(ctx, "session-relay", uint64(100+i))
		require.NoError(t, err)
	}

	// Verify counts
	retrieved, err := store.Get(ctx, "session-relay")
	require.NoError(t, err)
	require.Equal(t, int64(10), retrieved.RelayCount)
	// 100+101+102+...+109 = 1045
	require.Equal(t, uint64(1045), retrieved.TotalComputeUnits)
}

func TestRedisSessionStore_Close(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Close the store
	err := store.Close()
	require.NoError(t, err)

	// Save should fail after close
	snapshot := &SessionSnapshot{
		SessionID:               "session-after-close",
		SupplierOperatorAddress: "supplier1",
		State:                   SessionStateActive,
	}

	err = store.Save(ctx, snapshot)
	require.Error(t, err)
	require.Contains(t, err.Error(), "session store is closed")

	// Double close should be safe
	err = store.Close()
	require.NoError(t, err)
}

func TestRedisSessionStore_DefaultConfig(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	logger := polyzero.NewLogger()

	// Use empty config - should set defaults
	store := NewRedisSessionStore(logger, client, SessionStoreConfig{
		SupplierAddress: "supplier1",
	})
	defer store.Close()

	require.Equal(t, "ha:miner:sessions", store.config.KeyPrefix)
	require.Equal(t, 24*time.Hour, store.config.SessionTTL)
}

func TestRedisSessionStore_ClaimedRootHash(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	rootHash := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	snapshot := &SessionSnapshot{
		SessionID:               "session-with-hash",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateClaimed,
		ClaimedRootHash:         rootHash,
	}

	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// Verify root hash is preserved
	retrieved, err := store.Get(ctx, "session-with-hash")
	require.NoError(t, err)
	require.Equal(t, rootHash, retrieved.ClaimedRootHash)
}

func TestRedisSessionStore_SessionLifecycle(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestRedisSessionStore(t, "supplier1")

	// Simulate complete session lifecycle
	snapshot := &SessionSnapshot{
		SessionID:               "lifecycle-test",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		ApplicationAddress:      "app1",
		SessionStartHeight:      100,
		SessionEndHeight:        110,
		State:                   SessionStateActive,
	}

	// 1. Create active session
	err := store.Save(ctx, snapshot)
	require.NoError(t, err)

	// 2. Process some relays
	for i := 0; i < 5; i++ {
		err = store.IncrementRelayCount(ctx, "lifecycle-test", 100)
		require.NoError(t, err)
		err = store.UpdateWALPosition(ctx, "lifecycle-test", "wal-entry-"+string(rune('0'+i)))
		require.NoError(t, err)
	}

	// 3. Transition to claiming
	err = store.UpdateState(ctx, "lifecycle-test", SessionStateClaiming)
	require.NoError(t, err)

	// 4. Update with claimed root hash and transition to claimed
	retrieved, err := store.Get(ctx, "lifecycle-test")
	require.NoError(t, err)
	retrieved.ClaimedRootHash = []byte("root-hash-here")
	err = store.Save(ctx, retrieved)
	require.NoError(t, err)
	err = store.UpdateState(ctx, "lifecycle-test", SessionStateClaimed)
	require.NoError(t, err)

	// 5. Transition through proving to settled
	err = store.UpdateState(ctx, "lifecycle-test", SessionStateProving)
	require.NoError(t, err)
	err = store.UpdateState(ctx, "lifecycle-test", SessionStateSettled)
	require.NoError(t, err)

	// 6. Verify final state
	final, err := store.Get(ctx, "lifecycle-test")
	require.NoError(t, err)
	require.Equal(t, SessionStateSettled, final.State)
	require.Equal(t, int64(5), final.RelayCount)
	require.Equal(t, uint64(500), final.TotalComputeUnits)
	require.Equal(t, "wal-entry-4", final.LastWALEntryID)

	// 7. Clean up
	err = store.Delete(ctx, "lifecycle-test")
	require.NoError(t, err)
}
