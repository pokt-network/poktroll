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

func setupWALTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
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

func TestNewRedisWAL(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	require.NotNil(t, wal)
	require.Equal(t, "ha:miner:wal", wal.config.KeyPrefix)
	require.Equal(t, int64(DefaultWALMaxLen), wal.config.MaxLen)
}

func TestRedisWAL_Append(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-abc"

	entry := &WALEntry{
		RelayHash:    []byte("hash-123"),
		RelayBytes:   []byte("relay-data"),
		ServiceID:    "ethereum",
		ComputeUnits: 1,
	}

	id, err := wal.Append(ctx, sessionID, entry)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.Equal(t, id, entry.ID)

	// Verify size
	size, err := wal.Size(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, int64(1), size)
}

func TestRedisWAL_AppendBatch(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-batch"

	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), RelayBytes: []byte("relay-1"), ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), RelayBytes: []byte("relay-2"), ComputeUnits: 1},
		{RelayHash: []byte("hash-3"), RelayBytes: []byte("relay-3"), ComputeUnits: 1},
	}

	ids, err := wal.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)
	require.Len(t, ids, 3)

	for i, entry := range entries {
		require.Equal(t, ids[i], entry.ID)
	}

	size, err := wal.Size(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, int64(3), size)
}

func TestRedisWAL_AppendBatch_Empty(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()

	ids, err := wal.AppendBatch(ctx, "session", []*WALEntry{})
	require.NoError(t, err)
	require.Nil(t, ids)
}

func TestRedisWAL_ReadFrom(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-read"

	// Append entries
	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), RelayBytes: []byte("relay-1"), ServiceID: "eth", ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), RelayBytes: []byte("relay-2"), ServiceID: "eth", ComputeUnits: 2},
		{RelayHash: []byte("hash-3"), RelayBytes: []byte("relay-3"), ServiceID: "eth", ComputeUnits: 3},
	}

	ids, err := wal.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)

	// Read all from beginning
	readEntries, err := wal.ReadFrom(ctx, sessionID, "0")
	require.NoError(t, err)
	require.Len(t, readEntries, 3)

	for i, entry := range readEntries {
		require.Equal(t, entries[i].RelayHash, entry.RelayHash)
		require.Equal(t, entries[i].ServiceID, entry.ServiceID)
		require.Equal(t, entries[i].ComputeUnits, entry.ComputeUnits)
	}

	// Read from middle
	readEntries, err = wal.ReadFrom(ctx, sessionID, ids[0])
	require.NoError(t, err)
	require.Len(t, readEntries, 2) // Should skip first entry

	require.Equal(t, entries[1].RelayHash, readEntries[0].RelayHash)
	require.Equal(t, entries[2].RelayHash, readEntries[1].RelayHash)
}

func TestRedisWAL_ReadFrom_Empty(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()

	entries, err := wal.ReadFrom(ctx, "nonexistent-session", "0")
	require.NoError(t, err)
	require.Nil(t, entries)
}

func TestRedisWAL_Checkpoint(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-checkpoint"

	// Append entries
	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), ComputeUnits: 1},
	}
	ids, err := wal.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)

	// Set checkpoint at first entry
	err = wal.Checkpoint(ctx, sessionID, ids[0])
	require.NoError(t, err)

	// Get checkpoint
	checkpoint, err := wal.GetCheckpoint(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, ids[0], checkpoint)
}

func TestRedisWAL_GetCheckpoint_NotSet(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()

	checkpoint, err := wal.GetCheckpoint(ctx, "no-checkpoint-session")
	require.NoError(t, err)
	require.Equal(t, "0", checkpoint)
}

func TestRedisWAL_Trim(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-trim"

	// Append entries
	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), ComputeUnits: 1},
		{RelayHash: []byte("hash-3"), ComputeUnits: 1},
		{RelayHash: []byte("hash-4"), ComputeUnits: 1},
	}
	ids, err := wal.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)

	// Checkpoint at entry 2
	err = wal.Checkpoint(ctx, sessionID, ids[1])
	require.NoError(t, err)

	// Trim
	err = wal.Trim(ctx, sessionID)
	require.NoError(t, err)

	// Check remaining entries
	size, err := wal.Size(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, int64(3), size) // entries 2, 3, 4 remain

	// Verify we can still read from checkpoint
	remaining, err := wal.ReadFrom(ctx, sessionID, "0")
	require.NoError(t, err)
	require.Len(t, remaining, 3)
	require.Equal(t, entries[1].RelayHash, remaining[0].RelayHash)
}

func TestRedisWAL_DeleteSession(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()
	sessionID := "session-delete"

	// Append entries
	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), ComputeUnits: 1},
	}
	ids, err := wal.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)

	// Set checkpoint
	err = wal.Checkpoint(ctx, sessionID, ids[0])
	require.NoError(t, err)

	// Delete session
	err = wal.DeleteSession(ctx, sessionID)
	require.NoError(t, err)

	// Verify stream is empty
	size, err := wal.Size(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, int64(0), size)

	// Verify checkpoint is gone
	checkpoint, err := wal.GetCheckpoint(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, "0", checkpoint)
}

func TestRedisWAL_RecoveryScenario(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	sessionID := "session-recovery"
	ctx := context.Background()

	// Simulate first miner writing to WAL
	wal1 := NewRedisWAL(logger, client, config)

	entries := []*WALEntry{
		{RelayHash: []byte("hash-1"), ServiceID: "eth", ComputeUnits: 1},
		{RelayHash: []byte("hash-2"), ServiceID: "eth", ComputeUnits: 2},
		{RelayHash: []byte("hash-3"), ServiceID: "eth", ComputeUnits: 3},
	}
	ids, err := wal1.AppendBatch(ctx, sessionID, entries)
	require.NoError(t, err)

	// Checkpoint after first entry (simulating SMST flush)
	err = wal1.Checkpoint(ctx, sessionID, ids[0])
	require.NoError(t, err)

	// Miner1 crashes
	wal1.Close()

	// New miner takes over and recovers
	wal2 := NewRedisWAL(logger, client, config)
	defer wal2.Close()

	// Get last checkpoint
	checkpoint, err := wal2.GetCheckpoint(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, ids[0], checkpoint)

	// Read entries after checkpoint (need to replay these)
	toReplay, err := wal2.ReadFrom(ctx, sessionID, checkpoint)
	require.NoError(t, err)
	require.Len(t, toReplay, 2) // entries 2 and 3

	require.Equal(t, entries[1].RelayHash, toReplay[0].RelayHash)
	require.Equal(t, entries[2].RelayHash, toReplay[1].RelayHash)
}

func TestRedisWAL_EntryTimestamp(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()

	beforeAppend := time.Now()
	entry := &WALEntry{
		RelayHash:    []byte("hash"),
		ComputeUnits: 1,
	}

	_, err := wal.Append(ctx, "session", entry)
	require.NoError(t, err)

	// Timestamp should be set automatically
	require.False(t, entry.Timestamp.IsZero())
	require.True(t, entry.Timestamp.After(beforeAppend) || entry.Timestamp.Equal(beforeAppend))
}

func TestRedisWAL_Close_Safe(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)

	// Double close should be safe
	err := wal.Close()
	require.NoError(t, err)

	err = wal.Close()
	require.NoError(t, err)
}

func TestRedisWAL_MultipleSessionsIsolation(t *testing.T) {
	_, client := setupWALTestRedis(t)
	logger := polyzero.NewLogger()

	config := WALConfig{
		SupplierAddress: "pokt1supplier123",
	}

	wal := NewRedisWAL(logger, client, config)
	defer wal.Close()

	ctx := context.Background()

	// Append to session A
	entryA := &WALEntry{RelayHash: []byte("hash-A"), ComputeUnits: 1}
	_, err := wal.Append(ctx, "session-A", entryA)
	require.NoError(t, err)

	// Append to session B
	entryB := &WALEntry{RelayHash: []byte("hash-B"), ComputeUnits: 2}
	_, err = wal.Append(ctx, "session-B", entryB)
	require.NoError(t, err)

	// Read session A
	entriesA, err := wal.ReadFrom(ctx, "session-A", "0")
	require.NoError(t, err)
	require.Len(t, entriesA, 1)
	require.Equal(t, entryA.RelayHash, entriesA[0].RelayHash)

	// Read session B
	entriesB, err := wal.ReadFrom(ctx, "session-B", "0")
	require.NoError(t, err)
	require.Len(t, entriesB, 1)
	require.Equal(t, entryB.RelayHash, entriesB[0].RelayHash)

	// Delete session A shouldn't affect session B
	err = wal.DeleteSession(ctx, "session-A")
	require.NoError(t, err)

	entriesB, err = wal.ReadFrom(ctx, "session-B", "0")
	require.NoError(t, err)
	require.Len(t, entriesB, 1)
}
