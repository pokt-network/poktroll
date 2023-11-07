//go:generate mockgen -destination=../../internal/mocks/mockclient/events_query_client_mock.go -package=mockclient . Dialer,Connection,EventsQueryClient
//go:generate mockgen -destination=../../internal/mocks/mockclient/block_client_mock.go -package=mockclient . Block,BlockClient
//go:generate mockgen -destination=../../internal/mocks/mockclient/tx_client_mock.go -package=mockclient . TxContext,TxClient
//go:generate mockgen -destination=../../internal/mocks/mockclient/cosmos_tx_builder_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client TxBuilder
//go:generate mockgen -destination=../../internal/mocks/mockclient/cosmos_keyring_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/crypto/keyring Keyring
//go:generate mockgen -destination=../../internal/mocks/mockclient/cosmos_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client AccountRetriever

package client

import (
	"context"

	comettypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
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

// BlocksObservable is an observable which is notified with an either
// value which contains either an error or the event message bytes.
//
// TODO_HACK: The purpose of this type is to work around gomock's lack of
// support for generic types. For the same reason, this type cannot be an
// alias (i.e. EventsBytesObservable = observable.Observable[either.Either[[]byte]]).
type BlocksObservable observable.ReplayObservable[Block]

// BlockClient is an interface which provides notifications about newly committed
// blocks as well as direct access to the latest block via some blockchain API.
type BlockClient interface {
	// CommittedBlocksSequence returns an observable which emits newly committed blocks.
	CommittedBlocksSequence(context.Context) BlocksObservable
	// LatestBlock returns the latest block that has been committed.
	LatestBlock(context.Context) Block
	// Close unsubscribes all observers of the committed blocks sequence observable
	// and closes the events query client.
	Close()
}

// Block is an interface which abstracts the details of a block to its minimal
// necessary components.
type Block interface {
	Height() int64
	Hash() []byte
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
