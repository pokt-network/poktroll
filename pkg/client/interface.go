//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/events_query_client_mock.go -package=mockclient . Dialer,Connection,EventsQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/block_client_mock.go -package=mockclient . Block,BlockClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/delegation_client_mock.go -package=mockclient . DelegationClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/tx_client_mock.go -package=mockclient . TxContext,TxClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/supplier_client_mock.go -package=mockclient . SupplierClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/account_query_client_mock.go -package=mockclient . AccountQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/application_query_client_mock.go -package=mockclient . ApplicationQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/supplier_query_client_mock.go -package=mockclient . SupplierQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/session_query_client_mock.go -package=mockclient . SessionQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/shared_query_client_mock.go -package=mockclient . SharedQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/proof_query_client_mock.go -package=mockclient . ProofQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/service_query_client_mock.go -package=mockclient . ServiceQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/bank_query_client_mock.go -package=mockclient . BankQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_tx_builder_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client TxBuilder
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_keyring_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/crypto/keyring Keyring
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client AccountRetriever
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/comet_rpc_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client CometRPC

package client

import (
	"context"

	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// MsgCreateClaim is an interface satisfying proof.MsgCreateClaim concrete type
// used by the SupplierClient interface to avoid cyclic dependencies.
type MsgCreateClaim interface {
	cosmostypes.Msg
	GetRootHash() []byte
	GetSessionHeader() *sessiontypes.SessionHeader
	GetSupplierOperatorAddress() string
}

// MsgSubmitProof is an interface satisfying proof.MsgSubmitProof concrete type
// used by the SupplierClient interface to avoid cyclic dependencies.
type MsgSubmitProof interface {
	cosmostypes.Msg
	GetProof() []byte
	GetSessionHeader() *sessiontypes.SessionHeader
	GetSupplierOperatorAddress() string
}

// SupplierClient is an interface for sufficient for a supplier operator to be
// able to construct blockchain transactions from pocket protocol-specific messages
// related to its role.
type SupplierClient interface {
	// CreateClaims sends claim messages which creates an onchain commitment by
	// calling supplier to the given smt.SparseMerkleSumTree root hash of the given
	// session's mined relays.
	CreateClaims(
		ctx context.Context,
		claimMsgs ...MsgCreateClaim,
	) error
	// SubmitProof sends proof messages which contain the smt.SparseCompactMerkleClosestProof,
	// corresponding to some previously created claim for the same session.
	// The proof is validated onchain as part of the pocket protocol.
	SubmitProofs(
		ctx context.Context,
		sessionProofs ...MsgSubmitProof,
	) error
	// Address returns the operator address of the SupplierClient that will be submitting proofs & claims.
	OperatorAddress() *cosmostypes.AccAddress
}

// TxClient provides a synchronous interface initiating and waiting for transactions
// derived from cosmos-sdk messages, in a cosmos-sdk based blockchain network.
type TxClient interface {
	SignAndBroadcast(
		ctx context.Context,
		msgs ...cosmostypes.Msg,
	) either.AsyncError
}

// TxContext provides an interface which consolidates the operational dependencies
// required to facilitate the sender side of the transaction lifecycle: build, sign,
// encode, broadcast, and query (optional).
//
// TODO_IMPROVE: Avoid depending on cosmos-sdk structs or interfaces; add Pocket
// interface types to substitute:
//   - ResultTx
//   - TxResponse
//   - Keyring
//   - TxBuilder
type TxContext interface {
	// GetKeyring returns the associated key management mechanism for the transaction context.
	GetKeyring() cosmoskeyring.Keyring

	// NewTxBuilder creates and returns a new transaction builder instance.
	NewTxBuilder() cosmosclient.TxBuilder

	// SignTx signs a transaction using the specified key name. It can operate in offline mode,
	// and can overwrite any existing signatures based on the provided flags.
	SignTx(
		keyName string,
		txBuilder cosmosclient.TxBuilder,
		offline, overwriteSig bool,
	) error

	// EncodeTx takes a transaction builder and encodes it, returning its byte representation.
	EncodeTx(txBuilder cosmosclient.TxBuilder) ([]byte, error)

	// BroadcastTx broadcasts the given transaction to the network.
	BroadcastTx(txBytes []byte) (*cosmostypes.TxResponse, error)

	// QueryTx retrieves a transaction status based on its hash and optionally provides
	// proof of the transaction.
	QueryTx(
		ctx context.Context,
		txHash []byte,
		prove bool,
	) (*cometrpctypes.ResultTx, error)

	// GetClientCtx returns the cosmos-sdk client context associated with the transaction context.
	GetClientCtx() cosmosclient.Context

	// GetSimulatedTxGas returns the estimated gas for the given messages.
	GetSimulatedTxGas(
		ctx context.Context,
		signingKeyName string,
		msgs ...cosmostypes.Msg,
	) (uint64, error)
}

// Block is an interface which abstracts the details of a block to its minimal
// necessary components.
type Block interface {
	Height() int64
	Hash() []byte
	Txs() []comettypes.Tx
}

// EventsObservable is a replay observable for events of some type T.
// NB: This cannot be an alias due to gomock's lack of support for generic types.
type EventsObservable[T any] observable.ReplayObservable[T]

// EventsReplayClient is an interface which provides notifications about newly received
// events as well as direct access to the latest event via some blockchain API.
type EventsReplayClient[T any] interface {
	// EventsSequence returns an observable which emits new events.
	EventsSequence(context.Context) observable.ReplayObservable[T]
	// LastNEvents returns the latest N events that has been received.
	LastNEvents(ctx context.Context, n int) []T
	// Close unsubscribes all observers of the events sequence observable
	// and closes the events query client.
	Close()
}

// BlockReplayObservable is a defined type which is a replay observable of type Block.
// NB: This cannot be an alias due to gomock's lack of support for generic types.
type BlockReplayObservable EventsObservable[Block]

// BlockClient is an interface that wraps the EventsReplayClient interface
// specific for the EventsReplayClient[Block] implementation
type BlockClient interface {
	// CommittedBlocksSequence returns a BlockObservable that emits the
	// latest blocks that have been committed to the chain.
	CommittedBlocksSequence(context.Context) BlockReplayObservable
	// LastBlock returns the latest block that has been committed onchain.
	LastBlock(context.Context) Block
	// Close unsubscribes all observers of the committed block sequence
	// observable and closes the events query client.
	Close()
}

// RedelegationReplayObservable is a defined type which is a replay observable
// of type Redelegation.
// NB: This cannot be an alias due to gomock's lack of support for generic types.
type RedelegationReplayObservable EventsObservable[*apptypes.EventRedelegation]

// DelegationClient is an interface that wraps the EventsReplayClient interface
// specific for the EventsReplayClient[Redelegation] implementation
type DelegationClient interface {
	// RedelegationsSequence returns a Observable of Redelegations that
	// emits the latest redelegation events that have occurred on chain.
	RedelegationsSequence(context.Context) RedelegationReplayObservable
	// LastNRedelegations returns the latest N redelegation events that have
	// occurred on chain.
	LastNRedelegations(context.Context, int) []*apptypes.EventRedelegation
	// Close unsubscribes all observers of the committed block sequence
	// observable and closes the events query client.
	Close()
}

// EventsBytesObservable is an observable which is notified with an either
// value which contains either an error or the event message bytes.
//
// TODO_HACK: The purpose of this type is to work around gomock's lack of
// support for generic types. For the same reason, this type cannot be an
// alias (i.e. EventsBytesObservable = observable.Observable[either.Bytes]).
type EventsBytesObservable observable.Observable[either.Bytes]

// EventsQueryClient is used to subscribe to chain event messages matching the given query,
//
// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#Subscribe()` method for a similar purpose. Perhaps we could
// replace our custom implementation with one which wraps that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)
//
// NOTE: a branch which attempts this is available at:
// https://github.com/pokt-network/poktroll/pull/74
type EventsQueryClient interface {
	// EventsBytes returns an observable which is notified about chain event messages
	// matching the given query. It receives an either value which contains either an
	// error or the event message bytes.
	EventsBytes(
		ctx context.Context,
		query string,
	) (EventsBytesObservable, error)
	// Close unsubscribes all observers of each active query's events bytes
	// observable and closes the connection.
	Close()
}

// Connection is a transport agnostic, bi-directional, message-passing interface.
type Connection interface {
	// Receive blocks until a message is received or an error occurs.
	Receive() (msg []byte, err error)
	// Send sends a message and may return a synchronous error.
	Send(msg []byte) error
	// Close closes the connection.
	Close() error
}

// Dialer encapsulates the construction of connections.
type Dialer interface {
	// DialContext constructs a connection to the given URL and returns it or
	// potentially a synchronous error.
	DialContext(ctx context.Context, urlStr string) (Connection, error)
}

// EventsQueryClientOption defines a function type that modifies the EventsQueryClient.
type EventsQueryClientOption func(EventsQueryClient)

// TxClientOption defines a function type that modifies the TxClient.
type TxClientOption func(TxClient)

// SupplierClientOption defines a function type that modifies the SupplierClient.
type SupplierClientOption func(SupplierClient)

// DelegationClientOption defines a function type that modifies the DelegationClient.
type DelegationClientOption func(DelegationClient)

// BlockClientOption defines a function type that modifies the BlockClient.
type BlockClientOption func(BlockClient)

// EventsReplayClientOption defines a function type that modifies the ReplayClient.
type EventsReplayClientOption[T any] func(EventsReplayClient[T])

// AccountQueryClient defines an interface that enables the querying of the
// onchain account information
type AccountQueryClient interface {
	// GetAccount queries the chain for the details of the account provided
	GetAccount(ctx context.Context, address string) (cosmostypes.AccountI, error)

	// GetPubKeyFromAddress returns the public key of the given address.
	GetPubKeyFromAddress(ctx context.Context, address string) (cryptotypes.PubKey, error)
}

// ApplicationQueryClient defines an interface that enables the querying of the
// onchain application information
type ApplicationQueryClient interface {
	// GetApplication queries the chain for the details of the application provided
	GetApplication(ctx context.Context, appAddress string) (apptypes.Application, error)

	// GetAllApplications queries all onchain applications
	GetAllApplications(ctx context.Context) ([]apptypes.Application, error)

	// GetParams queries the chain for the application module parameters.
	GetParams(ctx context.Context) (*apptypes.Params, error)
}

// SupplierQueryClient defines an interface that enables the querying of the
// onchain supplier information
type SupplierQueryClient interface {
	// GetSupplier queries the chain for the details of the supplier provided
	GetSupplier(ctx context.Context, supplierOperatorAddress string) (sharedtypes.Supplier, error)
}

// SessionQueryClient defines an interface that enables the querying of the
// onchain session information
type SessionQueryClient interface {
	// GetSession queries the chain for the details of the session provided
	GetSession(
		ctx context.Context,
		appAddress string,
		serviceId string,
		blockHeight int64,
	) (*sessiontypes.Session, error)

	// GetParams queries the chain for the session module parameters.
	GetParams(ctx context.Context) (*sessiontypes.Params, error)
}

// SharedQueryClient defines an interface that enables the querying of the
// onchain shared module params.
type SharedQueryClient interface {
	// GetParams queries the chain for the current shared module parameters.
	GetParams(ctx context.Context) (*sharedtypes.Params, error)
	// GetSessionGracePeriodEndHeight returns the block height at which the grace period
	// for the session that includes queryHeight elapses.
	// The grace period is the number of blocks after the session ends during which relays
	// SHOULD be included in the session which most recently ended.
	GetSessionGracePeriodEndHeight(ctx context.Context, queryHeight int64) (int64, error)
	// GetClaimWindowOpenHeight returns the block height at which the claim window of
	// the session that includes queryHeight opens.
	GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error)
	// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
	// for the session that includes queryHeight can be committed for a given supplier.
	GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error)
	// GetProofWindowOpenHeight returns the block height at which the proof window of
	// the session that includes queryHeight opens.
	GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error)
	// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
	// for the session that includes queryHeight can be committed for a given supplier.
	GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error)
	// GetComputeUnitsToTokensMultiplier returns the multiplier used to convert compute units to tokens.
	GetComputeUnitsToTokensMultiplier(ctx context.Context) (uint64, error)
}

// BlockQueryClient defines an interface that enables the querying of
// onchain block information for a given height. If height is nil, the
// latest block is returned.
type BlockQueryClient interface {
	Block(ctx context.Context, height *int64) (*cometrpctypes.ResultBlock, error)
}

// ProofParams is a go interface type which corresponds to the poktroll.proof.Params
// protobuf message. Since the generated go types don't include interface types, this
// is necessary to prevent dependency cycles.
type ProofParams interface {
	GetProofRequestProbability() float64
	GetProofRequirementThreshold() *cosmostypes.Coin
	GetProofMissingPenalty() *cosmostypes.Coin
	GetProofSubmissionFee() *cosmostypes.Coin
}

// ProofQueryClient defines an interface that enables the querying of the
// onchain proof module params.
type ProofQueryClient interface {
	// GetParams queries the chain for the current proof module parameters.
	GetParams(ctx context.Context) (ProofParams, error)
}

// ServiceQueryClient defines an interface that enables the querying of the
// onchain service information
type ServiceQueryClient interface {
	// GetService queries the chain for the details of the service provided
	GetService(ctx context.Context, serviceId string) (sharedtypes.Service, error)
	GetServiceRelayDifficulty(ctx context.Context, serviceId string) (servicetypes.RelayMiningDifficulty, error)
}

// BankQueryClient defines an interface that enables the querying of the
// onchain bank information
type BankQueryClient interface {
	// GetBalance queries the chain for the uPOKT balance of the account provided
	GetBalance(ctx context.Context, address string) (*cosmostypes.Coin, error)
}

// KeyValueCache is a key/value store style interface for a cache of a single type.
// It is intended to be used to cache query responses (or derivatives thereof),
// where each key uniquely indexes the most recent query response.
type KeyValueCache[T any] interface {
	Get(key string) (T, error)
	Set(key string, value T) error
	Delete(key string)
	Clear()
}

// HistoricalKeyValueCache extends KeyValueCache to support getting and setting values
// at multiple heights for a given key.
type HistoricalKeyValueCache[T any] interface {
	KeyValueCache[T]
	// GetAsOfVersion retrieves the nearest value <= the specified version number.
	GetAsOfVersion(key string, version int64) (T, error)
	// SetAsOfVersion adds or updates a value at a specific version number.
	SetAsOfVersion(key string, value T, version int64) error
}
