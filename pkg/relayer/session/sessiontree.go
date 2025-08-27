package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/pokt-network/smt/kvstore/simplemap"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ relayer.SessionTree = (*sessionTree)(nil)

// sessionTree is an implementation of the SessionTree interface.
type sessionTree struct {
	logger polylog.Logger

	// sessionMu is a mutex used to protect sessionTree operations from concurrent access.
	sessionMu *sync.Mutex

	// sessionHeader is the header of the session corresponding to the SMST (Sparse Merkle State Trie).
	sessionHeader *sessiontypes.SessionHeader

	// sessionSMT is the SMST (Sparse Merkle State Trie) corresponding the session.
	sessionSMT smt.SparseMerkleSumTrie

	// supplierOperatorAddress is the address of the supplier's operator that owns this sessionTree.
	// RelayMiner can run suppliers for many supplier operator addresses at the same time,
	// and we need a way to group the session trees by the supplier operator address for that.
	supplierOperatorAddress string

	// claimedRoot is the root hash of the SMST needed for submitting the claim.
	// If it holds a non-nil value, it means that the SMST has been flushed,
	// committed to disk and no more updates can be made to it. A non-nil value also
	// indicates that a proof could be generated using ProveClosest function.
	claimedRoot []byte

	// proofPath is the path for which the proof was generated.
	proofPath []byte

	// compactProof is the generated compactProof for the session given a proofPath.
	compactProof *smt.SparseCompactMerkleClosestProof

	// compactProofBz is the marshaled proof for the session.
	compactProofBz []byte

	// treeStore is the KVStore used to store the SMST.
	// It can be either a pebble.PebbleKVStore (for disk persistence) or simplemap.SimpleMap (for in-memory).
	treeStore kvstore.MapStore

	// storePath is the path to the KVStore used to store the SMST.
	// It is created from the storePrefix and the session.sessionId.
	// We keep track of it so we can use it at the end of the claim/proof lifecycle
	// to delete the KVStore when it is no longer needed.
	storePath string

	isClaiming bool
}

// NewSessionTree creates a new sessionTree from a Session and a storePrefix. It also takes a function
// removeFromRelayerSessions that removes the sessionTree from the RelayerSessionsManager.
// It returns an error if the KVStore fails to be created.
func NewSessionTree(
	logger polylog.Logger,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddress string,
	storesDirectoryPath string,
) (relayer.SessionTree, error) {
	logger = logger.With(
		"session_id", sessionHeader.SessionId,
		"application_address", sessionHeader.ApplicationAddress,
		"service_id", sessionHeader.ServiceId,
		"supplier_operator_address", supplierOperatorAddress,
	)

	// Initialize the KVStore based on the storage type specified
	var treeStore kvstore.MapStore
	storePath := storesDirectoryPath // Store the original path for type checking

	switch storesDirectoryPath {
	case InMemoryStoreFilename:
		// SimpleMap in-memory storage (pure Go map)
		logger.Info().Msg("⚠️ Using SimpleMap in-memory store for session tree. Data will not be persisted on restart. ⚠️")
		treeStore = simplemap.NewSimpleMap()

	case InMemoryPebbleStoreFilename:
		// Pebble in-memory storage (with in-memory VFS)
		logger.Info().Msg("⚠️ Using Pebble in-memory store for session tree. Data will not be persisted on restart. ⚠️")
		pebbleStore, err := pebble.NewKVStore("") // Empty string triggers in-memory VFS in Pebble
		if err != nil {
			return nil, err
		}
		treeStore = pebbleStore

	default:
		// Disk-based persistent storage using Pebble
		// TODO_IMPROVE(#621): Use a single KV store instead of one per session for better RAM/IO efficiency.
		// KV databases are optimized for single database writes with prefixed keys.
		storePath = filepath.Join(storesDirectoryPath, supplierOperatorAddress, sessionHeader.SessionId)

		// Make sure storePath does not exist when creating a new SessionTree
		if _, err := os.Stat(storePath); err != nil && !os.IsNotExist(err) {
			return nil, ErrSessionTreeStorePathExists.Wrapf("storePath: %q", storePath)
		}
		logger.Info().Msgf("Using %s as the store path for session tree", storePath)

		pebbleStore, err := pebble.NewKVStore(storePath)
		if err != nil {
			return nil, err
		}
		treeStore = pebbleStore
	}
	logger = logger.With("store_path", storePath)

	// Create the SMST from the KVStore and a nil value hasher so the proof would
	// contain a non-hashed Relay that could be used to validate the proof onchain.
	trie := smt.NewSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), protocol.SMTValueHasher())

	sessionTree := &sessionTree{
		logger:                  logger,
		sessionHeader:           sessionHeader,
		storePath:               storePath,
		treeStore:               treeStore,
		sessionSMT:              trie,
		sessionMu:               &sync.Mutex{},
		supplierOperatorAddress: supplierOperatorAddress,
	}

	return sessionTree, nil
}

// importSessionTree reconstructs a previously created session tree from its persisted state on disk.
// This function handles two distinct scenarios:
// 1. Importing a claimed session (claim != nil): The tree is in a read-only state with a fixed root hash
// 2. Importing an unclaimed session (claim == nil): The tree is in a mutable state and can accept updates
//
// Returns a fully reconstructed SessionTree or an error if reconstruction fails
func importSessionTree(
	logger polylog.Logger,
	sessionSMT *prooftypes.SessionSMT,
	claim *prooftypes.Claim,
	storesDirectoryPath string,
) (relayer.SessionTree, error) {
	sessionId := sessionSMT.SessionHeader.SessionId
	supplierOperatorAddress := sessionSMT.SupplierOperatorAddress
	applicationAddress := sessionSMT.SessionHeader.ApplicationAddress
	serviceId := sessionSMT.SessionHeader.ServiceId
	smtRoot := sessionSMT.SmtRoot
	storePath := filepath.Join(storesDirectoryPath, supplierOperatorAddress, sessionId)

	// Verify the storage path exists - if not, the session data is missing or corrupted
	if _, err := os.Stat(storePath); err != nil {
		logger.Error().Err(err).Msgf("session tree store path does not exist: %q", storePath)
		return nil, err
	}

	// Initialize the basic session tree structure with metadata
	sessionTree := &sessionTree{
		logger:                  logger,
		sessionHeader:           sessionSMT.SessionHeader,
		storePath:               storePath,
		sessionMu:               &sync.Mutex{},
		supplierOperatorAddress: supplierOperatorAddress,
	}

	logger = logger.With(
		"service_id", serviceId,
		"application_address", applicationAddress,
		"supplier_operator_address", supplierOperatorAddress,
		"session_id", sessionId,
	)

	// SCENARIO 1: Session has been claimed
	// When a claim exists, the session tree is ready to be processed by the proof submission step.
	// The tree storage is not loaded immediately as it will only be needed if proof generation is requested.
	if claim != nil {
		sessionTree.claimedRoot = claim.RootHash
		sessionTree.isClaiming = true
		logger.Info().Msg("imported a session tree WITH A PREVIOUSLY COMMITTED onchain claim")
		return sessionTree, nil
	}

	// SCENARIO 2: Session has not been claimed
	// The session is still active and mutable, so we need to reconstruct the full tree
	// from the persisted storage to allow for additional relay updates.

	// Open the existing KVStore that contains the session's merkle tree data
	treeStore, err := pebble.NewKVStore(storePath)
	if err != nil {
		return nil, err
	}

	// Reconstruct the SMST from the persisted KVStore data using the previously saved root as the starting point
	trie := smt.ImportSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), smtRoot, protocol.SMTValueHasher())

	sessionTree.sessionSMT = trie
	sessionTree.treeStore = treeStore
	sessionTree.claimedRoot = nil  // explicitly set for posterity
	sessionTree.isClaiming = false // explicitly set for posterity

	logger.Info().Msg("imported a session tree WITHOUT A PREVIOUSLY COMMITTED onchain claim")

	return sessionTree, nil
}

// GetSession returns the session corresponding to the SMST.
func (st *sessionTree) GetSessionHeader() *sessiontypes.SessionHeader {
	return st.sessionHeader
}

// Update is a wrapper for the SMST's Update function. It updates the SMST with
// the given key, value, and weight.
// This function should be called by the Miner when a Relay has been successfully served.
// It returns an error if the SMST has been flushed to disk which indicates
// that updates are no longer allowed.
func (st *sessionTree) Update(key, value []byte, weight uint64) error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	if st.claimedRoot != nil {
		return ErrSessionTreeClosed
	}

	err := st.sessionSMT.Update(key, value, weight)
	if err != nil {
		return ErrSessionUpdatingTree.Wrapf("error: %v", err)
	}

	// DO NOT DELETE: Uncomment this for debugging and change to .Debug logs post MainNet.
	// count := st.sessionSMT.MustCount()
	// sum := st.sessionSMT.MustSum()
	// st.logger.Debug().Msgf("session tree updated and has count %d and sum %d", count, sum)

	return nil
}

// ProveClosest is a wrapper for the SMST's ProveClosest function. It returns a proof for the given path.
// This function is intended to be called after a session has been claimed and needs to be proven.
// If the proof has already been generated, it returns the cached proof.
// It returns an error if the SMST has not been flushed yet (the claim has not been generated)
func (st *sessionTree) ProveClosest(path []byte) (compactProof *smt.SparseCompactMerkleClosestProof, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// A claim need to be generated before a proof can be generated.
	if st.claimedRoot == nil {
		return nil, ErrSessionTreeNotClosed
	}

	// If the proof has already been generated, return the cached proof.
	if st.compactProof != nil {
		// Make sure the path is the same as the one for which the proof was generated.
		if !bytes.Equal(path, st.proofPath) {
			return nil, ErrSessionTreeProofPathMismatch
		}

		return st.compactProof, nil
	}

	// Handle restoration differently based on storage type
	var sessionSMT smt.SparseMerkleSumTrie

	switch st.storePath {
	case InMemoryStoreFilename:
		// SimpleMap: sessionSMT should still be available (preserved during Flush)
		if st.sessionSMT != nil {
			sessionSMT = st.sessionSMT
		} else {
			// Fallback: reimport from the active SimpleMap store
			sessionSMT = smt.ImportSparseMerkleSumTrie(st.treeStore, protocol.NewTrieHasher(), st.claimedRoot, protocol.SMTValueHasher())
		}

	case InMemoryPebbleStoreFilename:
		// Pebble in-memory: use preserved sessionSMT (can't restore from disk)
		if st.sessionSMT != nil {
			sessionSMT = st.sessionSMT
		} else {
			// Fallback: try to create fresh store (will likely fail for proofs)
			pebbleStore, pebbleErr := pebble.NewKVStore("") // Empty string for in-memory
			if pebbleErr != nil {
				return nil, pebbleErr
			}
			st.treeStore = pebbleStore
			sessionSMT = smt.ImportSparseMerkleSumTrie(st.treeStore, protocol.NewTrieHasher(), st.claimedRoot, protocol.SMTValueHasher())
		}

	default:
		// Disk-based persistent storage: restore KVStore from disk
		pebbleStore, pebbleErr := pebble.NewKVStore(st.storePath)
		if pebbleErr != nil {
			return nil, pebbleErr
		}
		st.treeStore = pebbleStore
		sessionSMT = smt.ImportSparseMerkleSumTrie(st.treeStore, protocol.NewTrieHasher(), st.claimedRoot, protocol.SMTValueHasher())
	}

	// Generate the proof and cache it along with the path for which it was generated.
	// There is no ProveClosest variant that generates a compact proof directly.
	// Generate a regular SparseMerkleClosestProof then compact it.
	proof, err := sessionSMT.ProveClosest(path)
	if err != nil {
		return nil, err
	}

	compactProof, err = smt.CompactClosestProof(proof, sessionSMT.Spec())
	if err != nil {
		return nil, err
	}

	compactProofBz, err := compactProof.Marshal()
	if err != nil {
		return nil, err
	}

	// If no error occurred, cache the proof and the path for which it was generated.
	st.sessionSMT = sessionSMT
	st.proofPath = path
	st.compactProof = compactProof
	st.compactProofBz = compactProofBz

	return st.compactProof, nil
}

// GetProofBz returns the marshaled proof for the session.
func (st *sessionTree) GetProofBz() []byte {
	return st.compactProofBz
}

// GetTrieSpec returns the trie spec of the SMST.
func (st *sessionTree) GetTrieSpec() smt.TrieSpec {
	return *st.sessionSMT.Spec()
}

// GetProof returns the proof for the SMST if it has been generated or nil otherwise.
func (st *sessionTree) GetProof() *smt.SparseCompactMerkleClosestProof {
	return st.compactProof
}

// Flush gets the root hash of the SMST needed for submitting the claim;
// then commits the entire tree to disk and stops the KVStore.
// It should be called before submitting the claim onchain. This function frees up the KVStore resources.
// If the SMST has already been flushed to disk, it returns the cached root hash.
func (st *sessionTree) Flush() (SMSTRoot []byte, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// We already have the root hash, return it.
	if st.claimedRoot != nil {
		return st.claimedRoot, nil
	}

	st.claimedRoot = st.sessionSMT.Root()

	// Handle different storage types during flush
	switch st.storePath {
	case InMemoryStoreFilename:
		// SimpleMap in-memory: keep everything active (no lifecycle management needed)
		st.logger.Debug().Msg("Not stopping SimpleMap session tree KVStore - keeping data in memory for proof generation.")
		// Keep both treeStore and sessionSMT references active for proof generation

	case InMemoryPebbleStoreFilename:
		// Pebble in-memory: preserve sessionSMT but stop the store
		// Unlike disk storage, we can't restore data from "disk" so we need to preserve the SMT
		if err := st.Stop(); err != nil {
			return nil, err
		}
		st.treeStore = nil
		// IMPORTANT: Keep sessionSMT for proof generation since we can't restore from disk
		st.logger.Debug().Msg("Stopped Pebble in-memory session tree KVStore - preserved sessionSMT for proof generation.")

	default:
		// Disk-based persistent storage: stop and clear references (data will be restored from disk)
		if err := st.Stop(); err != nil {
			return nil, err
		}
		st.treeStore = nil
		st.sessionSMT = nil
	}

	return st.claimedRoot, nil
}

// GetSMSTRoot returns the root hash of the SMST.
// This function is used to get the root hash of the SMST at any time.
// In particular, it is used to get the root hash of the SMST before it is flushed to disk
// for use-cases like restarting the relayer and resuming ongoing sessions.
func (st *sessionTree) GetSMSTRoot() (smtRoot smt.MerkleSumRoot) {
	if st.sessionSMT == nil {
		return nil
	}
	return st.sessionSMT.Root()
}

// GetClaimRoot returns the root hash of the SMST needed for creating the claim.
// It returns the root hash of the SMST only if the SMST has been flushed to disk.
func (st *sessionTree) GetClaimRoot() []byte {
	return st.claimedRoot
}

// Delete deletes the SMST from the KVStore and removes the sessionTree from the RelayerSessionsManager.
// WARNING: This function deletes the KVStore associated to the session and should be
// called only after the proof has been successfully submitted onchain and the servicer
// has confirmed that it has been rewarded.
func (st *sessionTree) Delete() error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()
	// After deletion, set treeStore to nil to:
	// - Prevent double-close operations.
	// - Avoid panics from future use of a closed kvstore instance.
	// - Signal that the treeStore is no longer valid.
	defer func() { st.treeStore = nil }()

	st.isClaiming = false

	// NB: We used to call `st.treeStore.ClearAll()` here.
	// This was intentionally removed to lower the IO load.
	// When the database is closed, it is deleted it from disk right away.

	// Handle stopping the KVStore if it's still active
	if st.treeStore != nil {
		switch st.storePath {
		case InMemoryStoreFilename:
			// SimpleMap stores don't need to be stopped

		case InMemoryPebbleStoreFilename:
			// Pebble in-memory stores need to be stopped
			if pebbleStore, ok := st.treeStore.(pebble.PebbleKVStore); ok {
				if err := pebbleStore.Stop(); err != nil {
					return err
				}
			}

		default:
			// Disk-based stores need to be stopped
			if pebbleStore, ok := st.treeStore.(pebble.PebbleKVStore); ok {
				if err := pebbleStore.Stop(); err != nil {
					return err
				}
			}
		}
	} else {
		st.logger.With(
			"claim_root", fmt.Sprintf("%x", st.GetClaimRoot()),
		).Info().Msg("KVStore is already stopped")
	}

	// Handle cleanup based on storage type
	switch st.storePath {
	case InMemoryStoreFilename:
		// SimpleMap: clear the data from memory
		st.logger.Info().Msg("Clearing SimpleMap in-memory session tree KVStore.")
		return st.treeStore.ClearAll()

	case InMemoryPebbleStoreFilename:
		// Pebble in-memory: data is already cleared when stopped
		st.logger.Info().Msg("Pebble in-memory session tree KVStore already cleared.")
		return nil

	default:
		// Disk-based persistent storage: delete the KVStore from disk
		st.logger.Info().Msgf("Deleting session tree KVStore from disk at %s", st.storePath)
		return os.RemoveAll(st.storePath)
	}
}

// StartClaiming marks the session tree as being picked up for claiming,
// so it won't be picked up by the relayer again.
// It returns an error if it has already been marked as such.
func (st *sessionTree) StartClaiming() error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	if st.isClaiming {
		return ErrSessionTreeAlreadyMarkedAsClaimed
	}

	st.isClaiming = true
	return nil
}

// GetSupplierOperatorAddress returns a stringified bech32 address of the supplier
// operator this sessionTree belongs to.
func (st *sessionTree) GetSupplierOperatorAddress() string {
	return st.supplierOperatorAddress
}

// Stop the KVStore and free up the in-memory resources used by the session tree.
// Calling Stop:
// - DOES NOT calculate the root hash of the SMST.
// - DOES commit the latest (current state) of the SMT to on-state storage.
func (st *sessionTree) Stop() error {
	if st.treeStore == nil {
		return nil
	}

	// Commit any pending changes to the KVStore before stopping it.
	if err := st.sessionSMT.Commit(); err != nil {
		return err
	}

	// Handle stopping based on storage type
	switch st.storePath {
	case InMemoryStoreFilename:
		// SimpleMap in-memory storage: no stopping required
		return nil

	case InMemoryPebbleStoreFilename:
		// Pebble in-memory storage: call Stop() if it's a PebbleKVStore
		if pebbleStore, ok := st.treeStore.(pebble.PebbleKVStore); ok {
			return pebbleStore.Stop()
		}
		return nil

	default:
		// Disk-based persistent storage: call Stop() if it's a PebbleKVStore
		if pebbleStore, ok := st.treeStore.(pebble.PebbleKVStore); ok {
			return pebbleStore.Stop()
		}
		return nil
	}
}
