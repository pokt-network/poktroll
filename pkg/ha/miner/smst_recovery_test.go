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

func newTestRecoverySetup(t *testing.T) (*SMSTSnapshotManager, *RedisSessionStore, *RedisWAL, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := polyzero.NewLogger()

	sessionStore := NewRedisSessionStore(logger, client, SessionStoreConfig{
		KeyPrefix:       "test:sessions",
		SupplierAddress: "supplier1",
		SessionTTL:      time.Hour,
	})

	wal := NewRedisWAL(logger, client, WALConfig{
		SupplierAddress: "supplier1",
		KeyPrefix:       "test:wal",
		MaxLen:          10000,
	})

	manager := NewSMSTSnapshotManager(logger, sessionStore, wal, SMSTRecoveryConfig{
		SupplierAddress: "supplier1",
		RecoveryTimeout: time.Minute,
	})

	t.Cleanup(func() {
		manager.Close()
		sessionStore.Close()
		wal.Close()
		client.Close()
		mr.Close()
	})

	return manager, sessionStore, wal, mr
}

func TestSMSTRecoveryService_RecoverEmptySessions(t *testing.T) {
	ctx := context.Background()
	manager, _, _, _ := newTestRecoverySetup(t)

	// Recover sessions when there are none
	sessions, err := manager.RecoverSessions(ctx)
	require.NoError(t, err)
	require.Nil(t, sessions)
}

func TestSMSTSnapshotManager_SessionLifecycle(t *testing.T) {
	ctx := context.Background()
	manager, sessionStore, _, _ := newTestRecoverySetup(t)

	// Create a session
	err := manager.OnSessionCreated(
		ctx,
		"session-123",
		"supplier1",
		"svc1",
		"app1",
		100,
		110,
	)
	require.NoError(t, err)

	// Verify session was created
	snapshot, err := sessionStore.Get(ctx, "session-123")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, SessionStateActive, snapshot.State)
	require.Equal(t, int64(0), snapshot.RelayCount)

	// Add some relays
	for i := 0; i < 5; i++ {
		relayHash := []byte("relay-hash-" + string(rune('0'+i)))
		relayBytes := []byte("relay-bytes-" + string(rune('0'+i)))
		err := manager.OnRelayMined(ctx, "session-123", relayHash, relayBytes, 100)
		require.NoError(t, err)
	}

	// Verify relay count updated
	snapshot, err = sessionStore.Get(ctx, "session-123")
	require.NoError(t, err)
	require.Equal(t, int64(5), snapshot.RelayCount)
	require.Equal(t, uint64(500), snapshot.TotalComputeUnits)
	require.NotEmpty(t, snapshot.LastWALEntryID)

	// Change state to claiming
	err = manager.OnSessionStateChange(ctx, "session-123", SessionStateClaiming)
	require.NoError(t, err)

	snapshot, err = sessionStore.Get(ctx, "session-123")
	require.NoError(t, err)
	require.Equal(t, SessionStateClaiming, snapshot.State)

	// Claim the session
	claimRoot := []byte("claim-root-hash-12345678")
	err = manager.OnSessionClaimed(ctx, "session-123", claimRoot)
	require.NoError(t, err)

	snapshot, err = sessionStore.Get(ctx, "session-123")
	require.NoError(t, err)
	require.Equal(t, SessionStateClaimed, snapshot.State)
	require.Equal(t, claimRoot, snapshot.ClaimedRootHash)

	// Settle the session
	err = manager.OnSessionSettled(ctx, "session-123")
	require.NoError(t, err)

	snapshot, err = sessionStore.Get(ctx, "session-123")
	require.NoError(t, err)
	require.Equal(t, SessionStateSettled, snapshot.State)
}

func TestSMSTRecoveryService_RecoverActiveSessions(t *testing.T) {
	ctx := context.Background()
	manager, sessionStore, wal, _ := newTestRecoverySetup(t)

	// Create multiple sessions in different states
	sessions := []struct {
		id    string
		state SessionState
	}{
		{"session-active", SessionStateActive},
		{"session-claiming", SessionStateClaiming},
		{"session-claimed", SessionStateClaimed},
		{"session-proving", SessionStateProving},
		{"session-settled", SessionStateSettled}, // Should not be recovered
	}

	for _, s := range sessions {
		// Create session snapshot
		snapshot := &SessionSnapshot{
			SessionID:               s.id,
			SupplierOperatorAddress: "supplier1",
			ServiceID:               "svc1",
			ApplicationAddress:      "app1",
			SessionStartHeight:      100,
			SessionEndHeight:        110,
			State:                   s.state,
		}
		err := sessionStore.Save(ctx, snapshot)
		require.NoError(t, err)

		// Add some WAL entries
		for i := 0; i < 3; i++ {
			entry := &WALEntry{
				SessionID:    s.id,
				RelayHash:    []byte("hash-" + s.id + "-" + string(rune('0'+i))),
				RelayBytes:   []byte("bytes-" + s.id + "-" + string(rune('0'+i))),
				ComputeUnits: uint64(100 + i),
			}
			_, err = wal.Append(ctx, s.id, entry)
			require.NoError(t, err)
		}
	}

	// Recover sessions
	recovered, err := manager.RecoverSessions(ctx)
	require.NoError(t, err)

	// Should recover 4 sessions (all except settled)
	require.Len(t, recovered, 4)

	// Verify each recovered session has WAL entries
	for _, r := range recovered {
		require.NotNil(t, r.Snapshot)
		require.Len(t, r.RelayUpdates, 3)

		// Verify WAL entries have the expected fields
		for _, entry := range r.RelayUpdates {
			require.NotEmpty(t, entry.RelayHash)
			require.NotEmpty(t, entry.RelayBytes)
			require.Greater(t, entry.ComputeUnits, uint64(0))
		}
	}
}

func TestSMSTRecoveryService_RecoverWithCheckpoint(t *testing.T) {
	ctx := context.Background()
	manager, sessionStore, wal, _ := newTestRecoverySetup(t)

	// Create a session
	snapshot := &SessionSnapshot{
		SessionID:               "session-checkpoint",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}
	err := sessionStore.Save(ctx, snapshot)
	require.NoError(t, err)

	// Add WAL entries
	var entryIDs []string
	for i := 0; i < 10; i++ {
		entry := &WALEntry{
			SessionID:    "session-checkpoint",
			RelayHash:    []byte("hash-" + string(rune('0'+i))),
			RelayBytes:   []byte("bytes-" + string(rune('0'+i))),
			ComputeUnits: 100,
		}
		id, err := wal.Append(ctx, "session-checkpoint", entry)
		require.NoError(t, err)
		entryIDs = append(entryIDs, id)
	}

	// Create a checkpoint at entry 5 (index 4)
	err = wal.Checkpoint(ctx, "session-checkpoint", entryIDs[4])
	require.NoError(t, err)

	// Recover sessions
	recovered, err := manager.RecoverSessions(ctx)
	require.NoError(t, err)
	require.Len(t, recovered, 1)

	// Should only have entries after the checkpoint (5 entries: indices 5-9)
	require.Len(t, recovered[0].RelayUpdates, 5)
}

func TestSMSTRecoveryService_RecoverSingleSession(t *testing.T) {
	ctx := context.Background()
	_, sessionStore, wal, mr := newTestRecoverySetup(t)

	// Create a recovery service directly for this test
	logger := polyzero.NewLogger()
	recovery := NewSMSTRecoveryService(logger, sessionStore, wal, SMSTRecoveryConfig{
		SupplierAddress: "supplier1",
		RecoveryTimeout: time.Minute,
	})
	defer recovery.Close()

	// Create a session
	snapshot := &SessionSnapshot{
		SessionID:               "single-session",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}
	err := sessionStore.Save(ctx, snapshot)
	require.NoError(t, err)

	// Add WAL entries
	for i := 0; i < 5; i++ {
		entry := &WALEntry{
			SessionID:    "single-session",
			RelayHash:    []byte("hash"),
			RelayBytes:   []byte("bytes"),
			ComputeUnits: 100,
		}
		_, err = wal.Append(ctx, "single-session", entry)
		require.NoError(t, err)
	}

	// Recover single session
	recovered, err := recovery.RecoverSession(ctx, "single-session")
	require.NoError(t, err)
	require.NotNil(t, recovered)
	require.Equal(t, "single-session", recovered.Snapshot.SessionID)
	require.Len(t, recovered.RelayUpdates, 5)

	// Should be cached
	cached, ok := recovery.GetRecoveredSession("single-session")
	require.True(t, ok)
	require.Equal(t, recovered, cached)

	// Clear and verify
	recovery.ClearRecoveredSession("single-session")
	_, ok = recovery.GetRecoveredSession("single-session")
	require.False(t, ok)

	// Test recovery of non-existent session
	_, err = recovery.RecoverSession(ctx, "nonexistent")
	require.Error(t, err)

	_ = mr // Keep miniredis reference
}

func TestSMSTSnapshotManager_Close(t *testing.T) {
	ctx := context.Background()
	manager, _, _, _ := newTestRecoverySetup(t)

	// Close the manager
	err := manager.Close()
	require.NoError(t, err)

	// Operations should fail after close
	err = manager.OnSessionCreated(ctx, "session", "supplier", "svc", "app", 100, 110)
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")

	err = manager.OnRelayMined(ctx, "session", []byte("hash"), []byte("bytes"), 100)
	require.Error(t, err)

	err = manager.OnSessionStateChange(ctx, "session", SessionStateClaimed)
	require.Error(t, err)

	// Double close should be safe
	err = manager.Close()
	require.NoError(t, err)
}

func TestSMSTRecoveryService_LargeBatchRecovery(t *testing.T) {
	ctx := context.Background()
	manager, sessionStore, wal, _ := newTestRecoverySetup(t)

	// Create a session with many WAL entries
	snapshot := &SessionSnapshot{
		SessionID:               "large-session",
		SupplierOperatorAddress: "supplier1",
		ServiceID:               "svc1",
		State:                   SessionStateActive,
	}
	err := sessionStore.Save(ctx, snapshot)
	require.NoError(t, err)

	// Add 250 WAL entries
	numEntries := 250
	for i := 0; i < numEntries; i++ {
		entry := &WALEntry{
			SessionID:    "large-session",
			RelayHash:    []byte("hash-" + string(rune(i%26+'a'))),
			RelayBytes:   []byte("bytes-data"),
			ComputeUnits: 100,
		}
		_, err = wal.Append(ctx, "large-session", entry)
		require.NoError(t, err)
	}

	// Recover sessions
	recovered, err := manager.RecoverSessions(ctx)
	require.NoError(t, err)
	require.Len(t, recovered, 1)
	require.Len(t, recovered[0].RelayUpdates, numEntries)
}
