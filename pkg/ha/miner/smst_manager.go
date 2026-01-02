package miner

import (
	"context"
	"fmt"
	"sync"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// InMemorySMSTManagerConfig contains configuration for the SMST manager.
type InMemorySMSTManagerConfig struct {
	// SupplierAddress is the supplier this manager is for.
	SupplierAddress string
}

// inMemorySMST holds an in-memory sparse merkle sum trie for a session.
type inMemorySMST struct {
	sessionID      string
	trie           smt.SparseMerkleSumTrie
	store          kvstore.MapStore
	claimedRoot    []byte
	proofPath      []byte
	compactProofBz []byte
	mu             sync.Mutex
}

// InMemorySMSTManager manages in-memory SMST trees for sessions.
// It implements the SMSTManager interface used by LifecycleCallback.
type InMemorySMSTManager struct {
	logger  polylog.Logger
	config  InMemorySMSTManagerConfig

	// Per-session SMST trees
	trees   map[string]*inMemorySMST
	treesMu sync.RWMutex
}

// NewInMemorySMSTManager creates a new in-memory SMST manager.
func NewInMemorySMSTManager(
	logger polylog.Logger,
	config InMemorySMSTManagerConfig,
) *InMemorySMSTManager {
	return &InMemorySMSTManager{
		logger: logging.ForSupplierComponent(logger, "smst_manager", config.SupplierAddress),
		config: config,
		trees:  make(map[string]*inMemorySMST),
	}
}

// GetOrCreateTree returns the SMST for a session, creating it if it doesn't exist.
func (m *InMemorySMSTManager) GetOrCreateTree(sessionID string) (*inMemorySMST, error) {
	m.treesMu.Lock()
	defer m.treesMu.Unlock()

	if tree, exists := m.trees[sessionID]; exists {
		return tree, nil
	}

	// Create a new in-memory tree
	store := simplemap.NewSimpleMap()
	trie := smt.NewSparseMerkleSumTrie(store, protocol.NewTrieHasher(), protocol.SMTValueHasher())

	tree := &inMemorySMST{
		sessionID: sessionID,
		trie:      trie,
		store:     store,
	}

	m.trees[sessionID] = tree

	m.logger.Debug().
		Str(logging.FieldSessionID, sessionID).
		Msg("created new in-memory SMST")

	return tree, nil
}

// UpdateTree adds a relay to the SMST for a session.
func (m *InMemorySMSTManager) UpdateTree(ctx context.Context, sessionID string, key, value []byte, weight uint64) error {
	tree, err := m.GetOrCreateTree(sessionID)
	if err != nil {
		return err
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	if tree.claimedRoot != nil {
		return fmt.Errorf("session %s has already been claimed, cannot update", sessionID)
	}

	if err := tree.trie.Update(key, value, weight); err != nil {
		return fmt.Errorf("failed to update SMST: %w", err)
	}

	return nil
}

// FlushTree flushes the SMST for a session and returns the root hash.
// After flushing, no more updates can be made to the tree.
func (m *InMemorySMSTManager) FlushTree(ctx context.Context, sessionID string) (rootHash []byte, err error) {
	m.treesMu.RLock()
	tree, exists := m.trees[sessionID]
	m.treesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	if tree.claimedRoot == nil {
		tree.claimedRoot = tree.trie.Root()
	}

	m.logger.Debug().
		Str(logging.FieldSessionID, sessionID).
		Int("root_hash_len", len(tree.claimedRoot)).
		Msg("flushed SMST")

	return tree.claimedRoot, nil
}

// GetTreeRoot returns the root hash for an already-flushed session.
func (m *InMemorySMSTManager) GetTreeRoot(ctx context.Context, sessionID string) (rootHash []byte, err error) {
	m.treesMu.RLock()
	tree, exists := m.trees[sessionID]
	m.treesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	if tree.claimedRoot != nil {
		return tree.claimedRoot, nil
	}

	// Return current root even if not flushed
	return tree.trie.Root(), nil
}

// ProveClosest generates a proof for the closest leaf to the given path.
func (m *InMemorySMSTManager) ProveClosest(ctx context.Context, sessionID string, path []byte) (proofBytes []byte, err error) {
	m.treesMu.RLock()
	tree, exists := m.trees[sessionID]
	m.treesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	// A tree needs to be flushed/claimed before generating a proof
	if tree.claimedRoot == nil {
		return nil, fmt.Errorf("session %s has not been claimed yet", sessionID)
	}

	// Generate the proof
	proof, err := tree.trie.ProveClosest(path)
	if err != nil {
		return nil, fmt.Errorf("failed to prove closest: %w", err)
	}

	// Compact the proof
	compactProof, err := smt.CompactClosestProof(proof, tree.trie.Spec())
	if err != nil {
		return nil, fmt.Errorf("failed to compact proof: %w", err)
	}

	// Marshal the proof
	proofBz, err := compactProof.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	// Cache the proof
	tree.proofPath = path
	tree.compactProofBz = proofBz

	m.logger.Debug().
		Str(logging.FieldSessionID, sessionID).
		Int("proof_len", len(proofBz)).
		Msg("generated proof")

	return proofBz, nil
}

// DeleteTree removes the SMST for a session.
func (m *InMemorySMSTManager) DeleteTree(ctx context.Context, sessionID string) error {
	m.treesMu.Lock()
	defer m.treesMu.Unlock()

	if _, exists := m.trees[sessionID]; !exists {
		return nil // Already deleted
	}

	delete(m.trees, sessionID)

	m.logger.Debug().
		Str(logging.FieldSessionID, sessionID).
		Msg("deleted SMST")

	return nil
}

// GetTreeCount returns the number of trees being managed.
func (m *InMemorySMSTManager) GetTreeCount() int {
	m.treesMu.RLock()
	defer m.treesMu.RUnlock()
	return len(m.trees)
}

// GetTreeStats returns statistics for a session tree.
func (m *InMemorySMSTManager) GetTreeStats(sessionID string) (count uint64, sum uint64, err error) {
	m.treesMu.RLock()
	tree, exists := m.trees[sessionID]
	m.treesMu.RUnlock()

	if !exists {
		return 0, 0, fmt.Errorf("session %s not found", sessionID)
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	count = tree.trie.MustCount()
	sum = tree.trie.MustSum()

	return count, sum, nil
}

// RebuildFromWAL rebuilds an SMST from WAL entries.
// This is used during recovery when a new leader takes over.
func (m *InMemorySMSTManager) RebuildFromWAL(ctx context.Context, sessionID string, entries []*WALEntry) error {
	tree, err := m.GetOrCreateTree(sessionID)
	if err != nil {
		return err
	}

	tree.mu.Lock()
	defer tree.mu.Unlock()

	for _, entry := range entries {
		if err := tree.trie.Update(entry.RelayHash, entry.RelayBytes, entry.ComputeUnits); err != nil {
			m.logger.Warn().
				Err(err).
				Str(logging.FieldSessionID, sessionID).
				Msg("failed to replay WAL entry, continuing")
			continue
		}
	}

	count := tree.trie.MustCount()
	m.logger.Info().
		Str(logging.FieldSessionID, sessionID).
		Int("wal_entries", len(entries)).
		Uint64("tree_count", count).
		Msg("rebuilt SMST from WAL")

	return nil
}

// Close cleans up all managed trees.
func (m *InMemorySMSTManager) Close() error {
	m.treesMu.Lock()
	defer m.treesMu.Unlock()

	m.trees = make(map[string]*inMemorySMST)

	m.logger.Info().Msg("SMST manager closed")
	return nil
}

// Ensure InMemorySMSTManager implements SMSTManager
var _ SMSTManager = (*InMemorySMSTManager)(nil)
