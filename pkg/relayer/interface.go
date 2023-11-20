//go:generate mockgen -destination=../../testutil/mockrelayer/relayer_proxy_mock.go -package=mockrelayer . RelayerProxy
//go:generate mockgen -destination=../../testutil/mockrelayer/miner_mock.go -package=mockrelayer . Miner
//go:generate mockgen -destination=../../testutil/mockrelayer/relayer_sessions_manager_mock.go -package=mockrelayer . RelayerSessionsManager

package relayer

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/observable"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TxClientContext is used to distinguish a cosmosclient.Context intended for use
// in transactions from others.
// This type is intentionally not an alias in order to make this distinction clear
// to the dependency injector
type TxClientContext client.Context

// RelaysObservable is an observable which is notified with Relay values.
//
// TODO_HACK: The purpose of this type is to work around gomock's lack of
// support for generic types. For the same reason, this type cannot be an
// alias (i.e. RelaysObservable = observable.Observable[*servicetypes.Relay]).
type RelaysObservable observable.Observable[*servicetypes.Relay]

// MinedRelaysObservable is an observable which is notified with MinedRelay values.
//
// TODO_HACK: The purpose of this type is to work around gomock's lack of
// support for generic types. For the same reason, this type cannot be an
// alias (i.e. MinedRelaysObservable = observable.Observable[*MinedRelay]).
type MinedRelaysObservable observable.Observable[*MinedRelay]

// Miner is responsible for observing servedRelayObs, hashing and checking the
// difficulty of each, finally publishing those with sufficient difficulty to
// minedRelayObs as they are applicable for relay volume.
type Miner interface {
	MinedRelays(
		ctx context.Context,
		servedRelayObs RelaysObservable,
	) (minedRelaysObs MinedRelaysObservable)
}

type MinerOption func(Miner)

// RelayerProxy is the interface for the proxy that serves relays to the application.
// It is responsible for starting and stopping all supported RelayServers.
// While handling requests and responding in a closed loop, it also notifies
// the miner about the relays that have been served.
type RelayerProxy interface {
	// Start starts all advertised relay servers and returns an error if any of them fail to start.
	Start(ctx context.Context) error

	// Stop stops all advertised relay servers and returns an error if any of them fail.
	Stop(ctx context.Context) error

	// ServedRelays returns an observable that notifies the miner about the relays that have been served.
	// A served relay is one whose RelayRequest's signature and session have been verified,
	// and its RelayResponse has been signed and successfully sent to the client.
	ServedRelays() RelaysObservable

	// VerifyRelayRequest is a shared method used by RelayServers to check the
	// relay request signature and session validity.
	// TODO_TECHDEBT(@red-0ne): This method should be moved out of the RelayerProxy interface
	// that should not be responsible for verifying relay requests.
	VerifyRelayRequest(
		ctx context.Context,
		relayRequest *servicetypes.RelayRequest,
		service *sharedtypes.Service,
	) error

	// SignRelayResponse is a shared method used by RelayServers to sign
	// and append the signature to the RelayResponse.
	// TODO_TECHDEBT(@red-0ne): This method should be moved out of the RelayerProxy interface
	// that should not be responsible for signing relay responses.
	SignRelayResponse(relayResponse *servicetypes.RelayResponse) error
}

type RelayerProxyOption func(RelayerProxy)

// RelayServer is the interface of the advertised relay servers provided by the RelayerProxy.
type RelayServer interface {
	// Start starts the service server and returns an error if it fails.
	Start(ctx context.Context) error

	// Stop terminates the service server and returns an error if it fails.
	Stop(ctx context.Context) error

	// Service returns the service to which the RelayServer relays.
	Service() *sharedtypes.Service
}

// RelayerSessionsManager is responsible for managing the relayer's session lifecycles.
// It handles the creation and retrieval of SMSTs (trees) for a given session, as
// well as the respective and subsequent claim creation and proof submission.
// This is largely accomplished by pipelining observables of relays and sessions
// through a series of map operations.
//
// TODO_TECHDEBT: add architecture diagrams covering observable flows throughout
// the relayer package.
type RelayerSessionsManager interface {
	// InsertRelays receives an observable of relays that should be included
	// in their respective session's SMST (tree).
	InsertRelays(minedRelaysObs MinedRelaysObservable)

	// Start iterates over the session trees at the end of each, respective, session.
	// The session trees are piped through a series of map operations which progress
	// them through the claim/proof lifecycle, broadcasting transactions to  the
	// network as necessary.
	Start(ctx context.Context)

	// Stop unsubscribes all observables from the InsertRelays observable which
	// will close downstream observables as they drain.
	//
	// TODO_TECHDEBT: Either add a mechanism to wait for draining to complete
	// and/or ensure that the state at each pipeline stage is persisted to disk
	// and exit as early as possible.
	Stop()
}

type RelayerSessionsManagerOption func(RelayerSessionsManager)

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
