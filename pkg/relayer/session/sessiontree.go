package session

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/badger"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/relayer"
	sessiontypes "github.com/pokt-network/poktroll/proto/types/session"
)

var _ relayer.SessionTree = (*sessionTree)(nil)

// sessionTree is an implementation of the SessionTree interface.
// TODO_TEST: Add tests to the sessionTree.
type sessionTree struct {
	// sessionMu is a mutex used to protect sessionTree operations from concurrent access.
	sessionMu *sync.Mutex

	// sessionHeader is the header of the session corresponding to the SMST (Sparse Merkle State Trie).
	sessionHeader *sessiontypes.SessionHeader

	// sessionSMT is the SMST (Sparse Merkle State Trie) corresponding the session.
	sessionSMT smt.SparseMerkleSumTrie

	// supplierAddress is the address of the supplier that owns this sessionTree.
	// RelayMiner can run suppliers for many supplier addresses at the same time,
	// and we need a way to group the session trees by the supplier address for that.
	supplierAddress *cosmostypes.AccAddress

	// claimedRoot is the root hash of the SMST needed for submitting the claim.
	// If it holds a non-nil value, it means that the SMST has been flushed,
	// committed to disk and no more updates can be made to it. A non-nil value also
	// indicates that a proof could be generated using ProveClosest function.
	claimedRoot []byte

	// proofPath is the path for which the proof was generated.
	proofPath []byte

	// proof is the generated proof for the session given a proofPath.
	proof *smt.SparseMerkleClosestProof

	// proofBz is the marshaled proof for the session.
	proofBz []byte

	// treeStore is the KVStore used to store the SMST.
	treeStore badger.BadgerKVStore

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
	supplierAddress *cosmostypes.AccAddress,
	storesDirectory string,
) (relayer.SessionTree, error) {
	// Join the storePrefix and the session.sessionId and supplier address to
	// create a unique storePath.
	storePath := filepath.Join(storesDirectory, sessionHeader.SessionId, "_", supplierAddress.String())

	// Make sure storePath does not exist when creating a new SessionTree
	if _, err := os.Stat(storePath); err != nil && !os.IsNotExist(err) {
		return nil, ErrSessionTreeStorePathExists.Wrapf("storePath: %q", storePath)
	}

	treeStore, err := badger.NewKVStore(storePath)
	if err != nil {
		return nil, err
	}

	// Create the SMST from the KVStore and a nil value hasher so the proof would
	// contain a non-hashed Relay that could be used to validate the proof on-chain.
	trie := smt.NewSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

	sessionTree := &sessionTree{
		sessionHeader:   sessionHeader,
		storePath:       storePath,
		treeStore:       treeStore,
		sessionSMT:      trie,
		sessionMu:       &sync.Mutex{},
		supplierAddress: supplierAddress,
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

	return st.sessionSMT.Update(key, value, weight)
}

// ProveClosest is a wrapper for the SMST's ProveClosest function. It returns a proof for the given path.
// This function is intended to be called after a session has been claimed and needs to be proven.
// If the proof has already been generated, it returns the cached proof.
// It returns an error if the SMST has not been flushed yet (the claim has not been generated)
// TODO_IMPROVE(#427): Compress the proof into a SparseCompactClosestMerkleProof
// prior to submitting to chain to reduce on-chain storage requirements for proofs.
func (st *sessionTree) ProveClosest(path []byte) (proof *smt.SparseMerkleClosestProof, err error) {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	// A claim need to be generated before a proof can be generated.
	if st.claimedRoot == nil {
		return nil, ErrSessionTreeNotClosed
	}

	// If the proof has already been generated, return the cached proof.
	if st.proof != nil {
		// Make sure the path is the same as the one for which the proof was generated.
		if !bytes.Equal(path, st.proofPath) {
			return nil, ErrSessionTreeProofPathMismatch
		}

		return st.proof, nil
	}

	// Restore the KVStore from disk since it has been closed after the claim has been generated.
	st.treeStore, err = badger.NewKVStore(st.storePath)
	if err != nil {
		return nil, err
	}

	sessionSMT := smt.ImportSparseMerkleSumTrie(st.treeStore, sha256.New(), st.claimedRoot, smt.WithValueHasher(nil))

	// Generate the proof and cache it along with the path for which it was generated.
	proof, err = sessionSMT.ProveClosest(path)
	if err != nil {
		return nil, err
	}

	proofBz, err := proof.Marshal()
	if err != nil {
		return nil, err
	}

	// If no error occurred, cache the proof and the path for which it was generated.
	st.sessionSMT = sessionSMT
	st.proofPath = path
	st.proof = proof
	st.proofBz = proofBz

	return st.proof, nil
}

// GetProofBz returns the marshaled proof for the session.
func (st *sessionTree) GetProofBz() []byte {
	return st.proofBz
}

// GetProof returns the proof for the SMST if it has been generated or nil otherwise.
func (st *sessionTree) GetProof() *smt.SparseMerkleClosestProof {
	return st.proof
}

// Flush gets the root hash of the SMST needed for submitting the claim;
// then commits the entire tree to disk and stops the KVStore.
// It should be called before submitting the claim on-chain. This function frees up the KVStore resources.
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
// called only after the proof has been successfully submitted on-chain and the servicer
// has confirmed that it has been rewarded.
func (st *sessionTree) Delete() error {
	st.sessionMu.Lock()
	defer st.sessionMu.Unlock()

	st.isClaiming = false

	if err := st.treeStore.ClearAll(); err != nil {
		return err
	}

	if err := st.treeStore.Stop(); err != nil {
		return err
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

// SupplierAddress returns a CosmosSDK address of the supplier this sessionTree belongs to.
func (st *sessionTree) GetSupplierAddress() *cosmostypes.AccAddress {
	return st.supplierAddress
}
