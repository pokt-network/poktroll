package session

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	sessiontypes "pocket/x/session/types"
	"sync"

	"github.com/pokt-network/smt"
)

var _ SessionTree = (*sessionTree)(nil)

// sessionTree is an implementation of the SessionTree interface.
type sessionTree struct {
	// session is the session corresponding to the SMST.
	session *sessiontypes.Session

	// tree is the SMST (Sparse Merkle State Tree) for the session.
	tree *smt.SMST

	// claimedRoot is the root hash of the SMST needed for submitting the claim.
	// If it holds a non-nil value, it means that the SMST has been flushed, committed to disk
	// and no more updates can be made to it. It also indicates that a proof could be generated
	// using ProveClosest by importing the SMST from disk.
	claimedRoot []byte

	// proofPath is the path for which the proof was generated.
	proofPath []byte

	// proof is the generated proof for the session.
	proof *smt.SparseMerkleClosestProof

	// treeStore is the KVStore used to store the SMST.
	treeStore smt.KVStore

	// storePath is the path to the KVStore used to store the SMST.
	// It is created from the storePrefix and the session.sessionId.
	// We store it so we can use it at the end of the claim/proof lifecycle
	// to delete the KVStore when it is no longer needed.
	storePath string

	// removeFromRelayerSessions is a function that removes the sessionTree from the sessionManager.
	// Since the sessionTree has no knowledge of the sessionManager, we pass this function
	// to the sessionTree so it can remove itself from the sessionManager when it is no longer needed.
	removeFromRelayerSessions func(session *sessiontypes.Session)

	// sessionMu is a mutex used to protect sessionTree operations from concurrent access.
	sessionMu *sync.Mutex
}

// NewSessionTree creates a new sessionTree from a session and a storePrefix. It also takes a function
// removeFromRelayerSessions that removes the sessionTree from the sessionManager.
// It returns an error if the KVStore fails to be created.
func NewSessionTree(
	session *sessiontypes.Session,
	storesDirectory string,
	removeFromRelayerSessions func(session *sessiontypes.Session),
) (SessionTree, error) {
	// join the storePrefix and the session.sessionId to create a unique storePath
	storePath := filepath.Join(storesDirectory, session.SessionId)

	treeStore, err := smt.NewKVStore(storePath)
	if err != nil {
		return nil, err
	}

	// create the SMST from the KVStore and a nil value hasher so the proof would contain a non-hashed Relay
	// that could be used to validate the proof on-chain.
	tree := smt.NewSparseMerkleSumTree(treeStore, sha256.New(), smt.WithValueHasher(nil))

	sessionTree := &sessionTree{
		session:   session,
		storePath: storePath,
		treeStore: treeStore,
		tree:      tree,

		removeFromRelayerSessions: removeFromRelayerSessions,
	}

	return sessionTree, nil
}

// GetSession returns the session corresponding to the SMST.
func (st *sessionTree) GetSession() *sessiontypes.Session {
	return st.session
}

// Update is a wrapper for the SMST's Update function. It updates the SMST with the given key, value, and weight.
// This function should be called when a Relay has been successfully served.
// It returns an error if the SMST has been flushed to disk and updates are no longer allowed.
func (st *sessionTree) Update(key, value []byte, weight uint64) error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	if st.claimedRoot != nil {
		return ErrSessionTreeClosed
	}

	return st.tree.Update(key, value, weight)
}

// ProveClosest is a wrapper for the SMST's ProveClosest function. It returns the proof for the given path.
// This function should be called when a session has been claimed and needs to be proven.
// If the proof has already been generated, it returns the cached proof.
// It returns an error if the SMST has not been flushed yet (the claim has not been generated)
func (st *sessionTree) ProveClosest(path []byte) (proof *smt.SparseMerkleClosestProof, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// A claim need to be generated before a proof can be generated.
	if st.claimedRoot == nil {
		return nil, ErrSessionTreeNotClosed
	}

	// If the proof has already been generated, return the cached proof.
	if st.proof != nil {
		return st.proof, nil
	}

	st.treeStore, err = smt.NewKVStore(st.storePath)
	if err != nil {
		return nil, err
	}

	st.tree = smt.ImportSparseMerkleSumTree(st.treeStore, sha256.New(), st.claimedRoot, smt.WithValueHasher(nil))

	// Generate the proof and cache it along with the path for which it was generated.
	st.proof, err = st.tree.ProveClosest(path)
	st.proofPath = path

	return st.proof, err
}

// Flush gets the root hash of the SMST needed for submitting the claim; then commits the entire tree to disk
// and stops the KVStore.
// It should be called before submitting the claim on-chain. This function frees up the in-memory resources.
// If the SMST has already been flushed to disk, it returns the cached root hash.
func (st *sessionTree) Flush() (SMSTRoot []byte, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// We already have the root hash, return it.
	if st.claimedRoot != nil {
		return st.claimedRoot, nil
	}

	st.claimedRoot = st.tree.Root()

	// Commit the tree to disk
	if err := st.tree.Commit(); err != nil {
		return nil, err
	}

	// Stop the KVStore
	if err := st.treeStore.Stop(); err != nil {
		return nil, err
	}

	st.treeStore = nil
	st.tree = nil

	return st.claimedRoot, nil
}

// Delete deletes the SMST from the KVStore and removes the sessionTree from the sessionManager.
func (st *sessionTree) Delete() error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	st.removeFromRelayerSessions(st.session)

	if err := st.treeStore.ClearAll(); err != nil {
		return err
	}

	if err := st.treeStore.Stop(); err != nil {
		return err
	}

	// Delete the KVStore from disk
	if err := os.RemoveAll(st.storePath); err != nil {
		return err
	}

	return nil
}
