package relayer

import (
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/observable"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// RelayerSessionsManager is an interface for managing the relayer's sessions and Sparse
// Merkle Sum Trees (SMSTs). It provides notifications about closing sessions that are
// ready to be claimed, and handles the creation and retrieval of SMSTs for a given session.
// It also handles the creation and retrieval of SMSTs for a given session.
type RelayerSessionsManager interface {
	// SessionsToClaim returns an observable that notifies of sessions ready to be claimed.
	SessionsToClaim() observable.Observable[SessionTree]

	// EnsureSessionTree returns the SMST (Sparse Merkle State Tree) for a given session header.
	// It is used to retrieve the SMST and update it when a Relay has been successfully served.
	// If the session header is seen for the first time, it creates a new SMST for it before returning it.
	// An error is returned if the corresponding KVStore for SMST fails to be created.
	EnsureSessionTree(sessionHeader *sessiontypes.SessionHeader) (SessionTree, error)
}

// SessionTree is an interface that wraps an SMST (Sparse Merkle State Tree) and its corresponding session.
type SessionTree interface {
	// GetSessionHeader returns the header of the session corresponding to the SMST.
	GetSessionHeader() *sessiontypes.SessionHeader

	// Update is a wrapper for the SMST's Update function. It updates the SMST with
	// the given key, value, and weight.
	// This function should be called when a Relay has been successfully served.
	Update(key, value []byte, weight uint64) error

	// ProveClosest is a wrapper for the SMST's ProveClosest function. It returns the
	// proof for the given path.
	// This function should be called several blocks after a session has been claimed and needs to be proven.
	ProveClosest(path []byte) (proof *smt.SparseMerkleClosestProof, err error)

	// Flush gets the root hash of the SMST needed for submitting the claim;
	// then commits the entire tree to disk and stops the KVStore.
	// It should be called before submitting the claim on-chain. This function frees up
	// the in-memory resources used by the SMST that are no longer needed while waiting
	// for the proof submission window to open.
	Flush() (SMSTRoot []byte, err error)

	// TODO_DISCUSS: This function should not be part of the interface as it is an optimization
	// aiming to free up KVStore resources after the proof is no longer needed.
	// Delete deletes the SMST from the KVStore.
	// WARNING: This function should be called only after the proof has been successfully
	// submitted on-chain and the servicer has confirmed that it has been rewarded.
	Delete() error
}
