//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockrelayer/relayer_proxy_mock.go -package=mockrelayer . RelayerProxy
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockrelayer/miner_mock.go -package=mockrelayer . Miner
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockrelayer/relayer_sessions_manager_mock.go -package=mockrelayer . RelayerSessionsManager
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockrelayer/relay_meter_mock.go -package=mockrelayer . RelayMeter

package relayer

import (
	"context"
	"net/http"

	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/observable"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

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

	// PingAll tests the connectivity between all the managed relay servers and their respective backend URLs.
	PingAll(ctx context.Context) error

	// Forward sends a request to the relay server that managed the given service id.
	Forward(ctx context.Context, serviceID string, w http.ResponseWriter, r *http.Request) error
}

type RelayerProxyOption func(RelayerProxy)

// RelayAuthenticator is the interface that authenticates the relay requests and
// responses (i.e. verifies the relay request signature and session validity, and
// signs the relay response).
type RelayAuthenticator interface {
	// VerifyRelayRequest verifies the relay request signature and session validity.
	VerifyRelayRequest(
		ctx context.Context,
		relayRequest *servicetypes.RelayRequest,
		serviceId string,
	) error

	// SignRelayResponse signs the relay response given a supplier operator address.
	SignRelayResponse(relayResponse *servicetypes.RelayResponse, supplierOperatorAddr string) error

	// GetSupplierOperatorAddresses returns the supplier operator addresses that
	// the relay authenticator can use to sign relay responses.
	GetSupplierOperatorAddresses() []string
}

type RelayAuthenticatorOption func(RelayAuthenticator)

// RelayServer is the interface of the advertised relay servers provided by the RelayerProxy.
type RelayServer interface {
	// Start starts the service server and returns an error if it fails.
	Start(ctx context.Context) error

	// Stop terminates the service server and returns an error if it fails.
	Stop(ctx context.Context) error

	// Ping tests the connection between the relay server and its backend URL.
	Ping(ctx context.Context) error

	// Forward sends a request to the supplier service.
	Forward(ctx context.Context, serviceID string, w http.ResponseWriter, r *http.Request) error
}

// RelayServers aggregates a slice of RelayServer interface.
type RelayServers []RelayServer

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

// SessionTree is an interface that wraps an SMST (Sparse Merkle State Trie) and its corresponding session.
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
	ProveClosest(path []byte) (proof *smt.SparseCompactMerkleClosestProof, err error)

	// GetClaimRoot returns the root hash of the SMST needed for creating the claim.
	GetClaimRoot() []byte

	// GetProofBz returns the proof created by ProveClosest needed for submitting
	// a proof in byte format.
	GetProofBz() []byte

	// Flush gets the root hash of the SMST needed for submitting the claim;
	// then commits the entire tree to disk and stops the KVStore.
	// It should be called before submitting the claim onchain. This function frees up
	// the in-memory resources used by the SMST that are no longer needed while waiting
	// for the proof submission window to open.
	Flush() (SMSTRoot []byte, err error)

	// TODO_DISCUSS: This function should not be part of the interface as it is an optimization
	// aiming to free up KVStore resources after the proof is no longer needed.
	// Delete deletes the SMST from the KVStore.
	// WARNING: This function should be called only after the proof has been successfully
	// submitted onchain and the servicer has confirmed that it has been rewarded.
	Delete() error

	// StartClaiming marks the session tree as being picked up for claiming,
	// so it won't be picked up by the relayer again.
	// It returns an error if it has already been marked as such.
	StartClaiming() error

	// GetSupplierOperatorAddress returns a stringified bech32 address of the supplier
	// operator this sessionTree belongs to.
	GetSupplierOperatorAddress() string

	// GetTrieSpec returns the trie spec of the SMST.
	GetTrieSpec() smt.TrieSpec
}

// RelayMeter is an interface that keeps track of the amount of stake consumed between
// a single onchain Application and a single onchain Supplier over the course of a single session.
// It enables the RelayMiner to rate limit the number of requests handled offchain as a function
// of the optimistic onchain rate limits.
type RelayMeter interface {
	// Start starts the relay meter.
	Start(ctx context.Context) error

	// AccumulateRelayReward adds the relay reward from the incoming request to session's accumulator.
	// The relay cost is added optimistically, assuming that the relay WILL be volume / reward applicable.
	//
	// The reason why optimistic AccumulateRelayReward + SetNonApplicableRelayReward is used instead of
	// a simpler AccumulateVolumeApplicableRelayReward is that when the relay is first seen
	// we don't know if it will be volume / reward applicable until it is served.
	//
	// To rate limit or not the current relay, we need to always optimistically account all relays as being
	// volume / reward applicable.
	AccumulateRelayReward(ctx context.Context, relayRequestMeta servicetypes.RelayRequestMetadata) error

	// SetNonApplicableRelayReward updates the relay meter for the given relay request as
	// non-applicable between a single Application and a single Supplier for a single session.
	// The volume / reward applicability of the relay is unknown to the relay miner
	// until the relay is served and the relay response signed.
	SetNonApplicableRelayReward(ctx context.Context, relayRequestMeta servicetypes.RelayRequestMetadata) error
}
