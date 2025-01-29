package session

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
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
	supplierOperatorAddress *cosmostypes.AccAddress

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
	treeStore pebble.PebbleKVStore

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
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddress *cosmostypes.AccAddress,
	storesDirectory string,
	logger polylog.Logger,
) (relayer.SessionTree, error) {
	// Join the storePrefix and the session.sessionId and supplier's operator address to
	// create a unique storePath.

	// TODO_IMPROVE(#621): instead of creating a new KV store for each session, it will be more beneficial to
	// use one key store. KV databases are often optimized for writing into one database. They keys can
	// use supplier address and session id as prefix. The current approach might not be RAM/IO efficient.
	storePath := filepath.Join(storesDirectory, supplierOperatorAddress.String(), sessionHeader.SessionId)

	// Make sure storePath does not exist when creating a new SessionTree
	if _, err := os.Stat(storePath); err != nil && !os.IsNotExist(err) {
		return nil, ErrSessionTreeStorePathExists.Wrapf("storePath: %q", storePath)
	}

	treeStore, err := pebble.NewKVStore(storePath)
	if err != nil {
		return nil, err
	}

	// Create the SMST from the KVStore and a nil value hasher so the proof would
	// contain a non-hashed Relay that could be used to validate the proof onchain.
	trie := smt.NewSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

	logger = logger.With(
		"store_path", storePath,
		"session_id", sessionHeader.SessionId,
		"supplier_operator_address", supplierOperatorAddress,
	)

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
	// fmt.Printf("Count: %d, Sum: %d\n", count, sum)

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

	// Restore the KVStore from disk since it has been closed after the claim has been generated.
	st.treeStore, err = pebble.NewKVStore(st.storePath)
	if err != nil {
		return nil, err
	}

	sessionSMT := smt.ImportSparseMerkleSumTrie(st.treeStore, sha256.New(), st.claimedRoot, smt.WithValueHasher(nil))

	// Generate the proof and cache it along with the path for which it was generated.
	// There is no ProveClosest variant that generates a compact proof directly.
	// Generate a regular SparseMerkleClosestProof then compact it.
	proof, err := sessionSMT.ProveClosest(path)
	if err != nil {
		return nil, err
	}

	compactProof, err = smt.CompactClosestProof(proof, &sessionSMT.TrieSpec)
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

	// Commit the tree to disk
	if err := st.sessionSMT.Commit(); err != nil {
		return nil, err
	}

	// Stop the KVStore
	if err := st.treeStore.Stop(); err != nil {
		return nil, err
	}

	st.treeStore = nil
	st.sessionSMT = nil

	return st.claimedRoot, nil
}

// GetClaimRoot returns the root hash of the SMST needed for creating the claim.
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

	st.isClaiming = false

	// NB: We used to call `st.treeStore.ClearAll()` here.
	// This was intentionally removed to lower the IO load.
	// When the database is closed, it is deleted it from disk right away.

	if st.treeStore != nil {
		if err := st.treeStore.Stop(); err != nil {
			return err
		}
	} else {
		st.logger.With(
			"claim_root", fmt.Sprintf("%x", st.GetClaimRoot()),
		).Info().Msg("KVStore is already stopped")
	}

	// Delete the KVStore from disk
	return os.RemoveAll(st.storePath)
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

// GetSupplierOperatorAddress returns a CosmosSDK address of the supplier this sessionTree belongs to.
func (st *sessionTree) GetSupplierOperatorAddress() *cosmostypes.AccAddress {
	return st.supplierOperatorAddress
}
