package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/smt/kvstore/simplemap"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ relayer.SessionTree = (*sessionTree)(nil)

const (
	minedRelaysWALDirectoryPath = "mined_relays_wal"
)

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

	// treeStore is the in-memory store used to store the SMST.
	treeStore kvstore.MapStore

	// minedRelaysWAL is the write-ahead log used to store the relays that have been mined
	// for this session. It is used to reconstruct the SMST in case of a crash or restart.
	minedRelaysWAL *minedRelaysWriteAheadLog

	isClaiming bool
}

// NewSessionTree creates a new sessionTree from a Session and a storePrefix. It also takes a function
// removeFromRelayerSessions that removes the sessionTree from the RelayerSessionsManager.
// It returns an error if the backing store fails to be created.
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

	treeStore := simplemap.NewSimpleMap()

	storePath := filepath.Join(storesDirectoryPath, minedRelaysWALDirectoryPath, supplierOperatorAddress, sessionHeader.SessionId)

	// Make sure storePath does not exist when creating a new SessionTree
	if _, err := os.Stat(storePath); err != nil && !os.IsNotExist(err) {
		return nil, ErrSessionTreeStorePathExists.Wrapf("storePath: %q", storePath)
	}
	logger.Info().Msgf("Using %s as the store path for session tree", storePath)

	// Update the logger with the store path
	logger = logger.With("store_path", storePath)

	// Create the SMST from the in-memory store and a nil value hasher so the proof would
	// contain a non-hashed Relay that could be used to validate the proof onchain.
	trie := smt.NewSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), protocol.SMTValueHasher())

	minedRelaysWAL, err := NewMinedRelaysWriteAheadLog(storePath, logger)
	if err != nil {
		return nil, err
	}

	// Create the sessionTree
	sessionTree := &sessionTree{
		logger:                  logger,
		sessionHeader:           sessionHeader,
		minedRelaysWAL:          minedRelaysWAL,
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
	storePath := filepath.Join(storesDirectoryPath, supplierOperatorAddress, sessionId)

	// Verify the storage path exists - if not, the session data is missing or corrupted
	if _, err := os.Stat(storePath); err != nil {
		logger.Error().Err(err).Msgf("session tree store path does not exist: %q", storePath)
		return nil, err
	}

	logger = logger.With("store_path", storePath)

	// Reconstruct the SMST from the persisted WAL
	logger.Info().Msg("Reconstructing the session SMT from the persisted WAL")
	treeStore := simplemap.NewSimpleMap()
	trie, err := reconstructSMTFromMinedRelaysLog(storePath, treeStore, logger)
	if err != nil {
		return nil, err
	}

	minedRelaysWAL, err := NewMinedRelaysWriteAheadLog(storePath, logger)
	if err != nil {
		return nil, err
	}

	// Initialize the basic session tree structure with metadata
	sessionTree := &sessionTree{
		logger:                  logger,
		sessionHeader:           sessionSMT.SessionHeader,
		minedRelaysWAL:          minedRelaysWAL,
		sessionMu:               &sync.Mutex{},
		supplierOperatorAddress: supplierOperatorAddress,
		sessionSMT:              trie,
		treeStore:               treeStore,
	}

	logger = logger.With(
		"service_id", serviceId,
		"application_address", applicationAddress,
		"supplier_operator_address", supplierOperatorAddress,
		"session_id", sessionId,
	)

	// SCENARIO 1: Session has been claimed
	// When a claim exists, the session tree is ready to be processed by the proof submission step.
	if claim != nil {
		sessionTree.claimedRoot = claim.RootHash
		sessionTree.isClaiming = true
		logger.Info().Msg("imported a session tree WITH A PREVIOUSLY COMMITTED onchain claim")
		return sessionTree, nil
	}

	// SCENARIO 2: Session has not been claimed, it is still active and mutable.
	//
	// When importing a session without an onchain claim, DO NOT set the claimedRoot.
	// Setting claimedRoot blocks further updates via the Update() method.
	//
	// The session may have been flushed during shutdown, but we still want to allow
	// updates after restart for sessions that haven't been claimed yet.
	// The claim creation process will call Flush() when needed, which will set
	// the claimedRoot at the appropriate time.
	//
	// This design allows:
	// 1. Sessions to continue accumulating relays after restart
	// 2. The claim creation process to work correctly by calling Flush()
	// 3. The same root to be generated since the imported SMT has the persisted state
	sessionTree.claimedRoot = nil  // Keep nil to allow updates
	sessionTree.isClaiming = false // Not yet in the claiming pipeline

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

	// Append the mined relay to the write-ahead log to ensure recovery in case of crashes or restarts.
	st.minedRelaysWAL.AppendMinedRelay(key, value, weight)

	if err := st.sessionSMT.Update(key, value, weight); err != nil {
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

	logger := st.logger.With("method", "ProveClosest")

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

	// Restore the sessionSMT from the persisted or in-memory storage.
	if st.sessionSMT == nil {
		logger.Error().Msg("🚨 SHOULD RARELY HAPPEN: sessionSMT is NULL after restoration attempt! Cannot generate proof! Claims exist but REWARDS WILL BE LOST! 🚨")
		return nil, fmt.Errorf("sessionSMT is nil - cannot generate proof for session %s", st.sessionHeader.SessionId)
	}

	// Generate the proof and cache it along with the path for which it was generated.
	// There is no ProveClosest variant that generates a compact proof directly.
	// Generate a regular SparseMerkleClosestProof then compact it.
	proof, err := st.sessionSMT.ProveClosest(path)
	if err != nil {
		logger.Error().Err(err).Msg("🚨 SHOULD RARELY HAPPEN: Proving the path in the sessionSMT failed! Cannot generate proof! Claims exist but REWARDS WILL BE LOST! 🚨")
		return nil, err
	}

	compactProof, err = smt.CompactClosestProof(proof, st.sessionSMT.Spec())
	if err != nil {
		return nil, err
	}

	compactProofBz, err := compactProof.Marshal()
	if err != nil {
		return nil, err
	}

	// If no error occurred, cache the proof and the path for which it was generated.
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
// It should be called before submitting the claim onchain.
// If the SMST has already been flushed to disk, it returns the cached root hash.
func (st *sessionTree) Flush() (SMSTRoot []byte, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	if st.claimedRoot == nil {
		st.claimedRoot = st.sessionSMT.Root()
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

// Close gracefully releases runtime resources associated with this session tree
// without deleting any persisted evidence.
// It ensures any buffered mined relay entries in the write-ahead log (WAL) are
// flushed to disk and closes the underlying WAL file handle.
func (st *sessionTree) Close() error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// If the WAL is already closed or was never created, there is nothing to do.
	if st.minedRelaysWAL == nil {
		return nil
	}

	if err := st.minedRelaysWAL.Close(); err != nil {
		st.logger.Error().Err(err).Msg("Failed to close the minedRelaysWAL")
		return err
	}

	st.minedRelaysWAL = nil

	return nil
}

// Delete deletes the SMST WAL and removes the sessionTree from the RelayerSessionsManager.
// WARNING: This function deletes the WAL associated to the session and should be
// called only after the proof has been successfully submitted onchain and the servicer
// has confirmed that it has been rewarded.
func (st *sessionTree) Delete() error {
	logger := st.logger.With("method", "Delete")

	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	st.isClaiming = false

	if err := st.treeStore.ClearAll(); err != nil {
		logger.Error().Err(err).Msg("Failed to clear SimpleMap treeStore")
		return err
	}

	if st.minedRelaysWAL == nil {
		logger.Warn().Msg("minedRelaysWAL is nil, nothing to close or remove")
		return nil
	}

	// Close and remove the WAL directory
	if err := st.minedRelaysWAL.CloseAndRemove(); err != nil {
		logger.Error().Err(err).Msg("Failed to remove session tree store directory")
		return err
	}

	st.minedRelaysWAL = nil

	return nil
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
