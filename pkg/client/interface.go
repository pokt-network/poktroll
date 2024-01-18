//go:generate mockgen -destination=../../testutil/mockclient/events_query_client_mock.go -package=mockclient . Dialer,Connection,EventsQueryClient
//go:generate mockgen -destination=../../testutil/mockclient/block_client_mock.go -package=mockclient . Block,BlockClient
//go:generate mockgen -destination=../../testutil/mockclient/delegation_client_mock.go -package=mockclient . Redelegation,DelegationClient
//go:generate mockgen -destination=../../testutil/mockclient/tx_client_mock.go -package=mockclient . TxContext,TxClient
//go:generate mockgen -destination=../../testutil/mockclient/supplier_client_mock.go -package=mockclient . SupplierClient
//go:generate mockgen -destination=../../testutil/mockclient/account_query_client_mock.go -package=mockclient . AccountQueryClient
//go:generate mockgen -destination=../../testutil/mockclient/application_query_client_mock.go -package=mockclient . ApplicationQueryClient
//go:generate mockgen -destination=../../testutil/mockclient/supplier_query_client_mock.go -package=mockclient . SupplierQueryClient
//go:generate mockgen -destination=../../testutil/mockclient/session_query_client_mock.go -package=mockclient . SessionQueryClient
//go:generate mockgen -destination=../../testutil/mockclient/cosmos_tx_builder_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client TxBuilder
//go:generate mockgen -destination=../../testutil/mockclient/cosmos_keyring_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/crypto/keyring Keyring
//go:generate mockgen -destination=../../testutil/mockclient/cosmos_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client AccountRetriever

package client

import (
	"context"

	comettypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SupplierClient is an interface for sufficient for a supplier operator to be
// able to construct blockchain transactions from pocket protocol-specific messages
// related to its role.
type SupplierClient interface {
	// CreateClaim sends a claim message which creates an on-chain commitment by
	// calling supplier to the given smt.SparseMerkleSumTree root hash of the given
	// session's mined relays.
	CreateClaim(
		ctx context.Context,
		sessionHeader sessiontypes.SessionHeader,
		rootHash []byte,
	) error
	// SubmitProof sends a proof message which contains the
	// smt.SparseMerkleClosestProof, corresponding to some previously created claim
	// for the same session. The proof is validated on-chain as part of the pocket
	// protocol.
	SubmitProof(
		ctx context.Context,
		sessionHeader sessiontypes.SessionHeader,
		proof *smt.SparseMerkleClosestProof,
	) error
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
	) (*comettypes.ResultTx, error)
}

// Block is an interface which abstracts the details of a block to its minimal
// necessary components.
type Block interface {
	Height() int64
	Hash() []byte
}

// Redelegation is an interface which wraps the EventRedelegation event
// emitted by the application module.
// See: proto/pocket/application/types/event.proto#EventRedelegation
type Redelegation interface {
	GetAppAddress() string
	GetGatewayAddress() string
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
	// LastNBlocks returns the latest N blocks that have been committed to
	// the chain.
	LastNBlocks(context.Context, int) []Block
	// Close unsubscribes all observers of the committed block sequence
	// observable and closes the events query client.
	Close()
}

// RedelegationReplayObservable is a defined type which is a replay observable
// of type Redelegation.
// NB: This cannot be an alias due to gomock's lack of support for generic types.
type RedelegationReplayObservable EventsObservable[Redelegation]

// DelegationClient is an interface that wraps the EventsReplayClient interface
// specific for the EventsReplayClient[Redelegation] implementation
type DelegationClient interface {
	// RedelegationsSequence returns a Observable of Redelegations that
	// emits the latest redelegation events that have occurred on chain.
	RedelegationsSequence(context.Context) RedelegationReplayObservable
	// LastNRedelegations returns the latest N redelegation events that have
	// occurred on chain.
	LastNRedelegations(context.Context, int) []Redelegation
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

// AccountQueryClient defines an interface that enables the querying of the
// on-chain account information
type AccountQueryClient interface {
	// GetAccount queries the chain for the details of the account provided
	GetAccount(ctx context.Context, address string) (accounttypes.AccountI, error)
}

// ApplicationQueryClient defines an interface that enables the querying of the
// on-chain application information
type ApplicationQueryClient interface {
	// GetApplication queries the chain for the details of the application provided
	GetApplication(ctx context.Context, appAddress string) (apptypes.Application, error)
}

// SupplierQueryClient defines an interface that enables the querying of the
// on-chain supplier information
type SupplierQueryClient interface {
	// GetSupplier queries the chain for the details of the supplier provided
	GetSupplier(ctx context.Context, supplierAddress string) (sharedtypes.Supplier, error)
}

// SessionQueryClient defines an interface that enables the querying of the
// on-chain session information
type SessionQueryClient interface {
	// GetSession queries the chain for the details of the session provided
	GetSession(
		ctx context.Context,
		appAddress string,
		serviceId string,
		blockHeight int64,
	) (*sessiontypes.Session, error)
}
