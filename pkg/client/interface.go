//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/block_client_mock.go -package=mockclient . Block,BlockClient
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
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/bank_grpc_query_client_mock.go -package=mockclient . BankGRPCQueryClient
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_tx_builder_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client TxBuilder
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_keyring_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/crypto/keyring Keyring
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/cosmos_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client AccountRetriever
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/comet_rpc_client_mock.go -package=mockclient github.com/cosmos/cosmos-sdk/client CometRPC
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/comet_client_mock.go -package=mockclient github.com/cometbft/cometbft/rpc/client Client
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/grpc_client_conn_mock.go -package=mockclient github.com/cosmos/gogoproto/grpc ClientConn
//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockclient/signing_tx.go -package=mockclient github.com/cosmos/cosmos-sdk/x/auth/signing Tx

package client

import (
	"context"

	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/hashicorp/go-version"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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
	// CreateClaims sends claim messages which creates an onchain commitment by calling supplier
	// to the given smt.SparseMerkleSumTree root hash of the given session's mined relays.
	//
	// A timeoutHeight is provided to ensure that the claim is not created after the claim
	// window has closed.
	CreateClaims(
		ctx context.Context,
		timeoutHeight int64,
		claimMsgs ...MsgCreateClaim,
	) error

	// SubmitProof sends proof messages which contain the smt.SparseCompactMerkleClosestProof,
	// corresponding to some previously created claim for the same session.
	// The proof is validated onchain as part of the pocket protocol.
	//
	// A timeoutHeight is provided to ensure that the proof is not submitted after
	// the proof window has closed.
	SubmitProofs(
		ctx context.Context,
		timeoutHeight int64,
		sessionProofs ...MsgSubmitProof,
	) error
	// OperatorAddress returns the bech32 string representation of the supplier operator address.
	OperatorAddress() string
}

// TxClient provides a synchronous interface initiating and waiting for transactions
// derived from cosmos-sdk messages, in a cosmos-sdk based blockchain network.
type TxClient interface {
	SignAndBroadcastWithTimeoutHeight(
		ctx context.Context,
		timeoutHeight int64,
		msgs ...cosmostypes.Msg,
	) (txResponse *cosmostypes.TxResponse, eitherErr either.AsyncError)

	SignAndBroadcast(
		ctx context.Context,
		msgs ...cosmostypes.Msg,
	) (txResponse *cosmostypes.TxResponse, eitherErr either.AsyncError)
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
		offline, overwriteSig, unordered bool,
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

	// WithUnordered returns a copy of the transaction context with the unordered flag set.
	WithUnordered(bool) TxContext
}

// Block is an interface which abstracts the details of a block to its minimal
// necessary components.
type Block interface {
	Height() int64
	Hash() []byte
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

	// GetChainVersion returns the current chain version.
	GetChainVersion() *version.Version
}

// TxClientOption defines a function type that modifies the TxClient.
type TxClientOption func(TxClient)

// SupplierClientOption defines a function type that modifies the SupplierClient.
type SupplierClientOption func(SupplierClient)

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

	// GetParams queries the chain for the supplier module parameters.
	GetParams(ctx context.Context) (*suppliertypes.Params, error)
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
}

// BlockQueryClient defines an interface that enables the querying of
// onchain block information for a given height. If height is nil, the
// latest block is returned.
type BlockQueryClient interface {
	Block(ctx context.Context, height *int64) (*cometrpctypes.ResultBlock, error)
}

// ProofParams is a go interface type reflecting the pocket.proof.Params protobuf.
// This is necessary since to prevent dependency cycles since generated go types don't interface types.
type ProofParams interface {
	GetProofRequestProbability() float64
	GetProofRequirementThreshold() *cosmostypes.Coin
	GetProofMissingPenalty() *cosmostypes.Coin
	GetProofSubmissionFee() *cosmostypes.Coin
}

// Claim is a go interface type reflecting the pocket.proof.Claim protobuf.
// This is necessary since to prevent dependency cycles since generated go types don't interface types.
type Claim interface {
	GetSupplierOperatorAddress() string
	GetSessionHeader() *sessiontypes.SessionHeader
	GetRootHash() []byte
}

// ProofQueryClient defines an interface that enables the querying of the
// onchain proof module params.
type ProofQueryClient interface {
	// GetParams queries the chain for the current proof module parameters.
	GetParams(ctx context.Context) (ProofParams, error)

	// GetClaim queries the chain for the full claim associatd with the (supplier, sessionId).
	GetClaim(ctx context.Context, supplierOperatorAddress string, sessionId string) (Claim, error)
}

// ServiceQueryClient defines an interface that enables the querying of the
// onchain service information
type ServiceQueryClient interface {
	// GetService queries the chain for the details of the service provided
	GetService(ctx context.Context, serviceId string) (sharedtypes.Service, error)
	GetServiceRelayDifficulty(ctx context.Context, serviceId string) (servicetypes.RelayMiningDifficulty, error)
	// GetParams queries the chain for the current proof module parameters.
	GetParams(ctx context.Context) (*servicetypes.Params, error)
}

// BankQueryClient defines an interface that enables the querying of the
// onchain bank information
type BankQueryClient interface {
	// GetBalance queries the chain for the uPOKT balance of the account provided
	GetBalance(ctx context.Context, address string) (*cosmostypes.Coin, error)
}

type BankGRPCQueryClient interface {
	AllBalances(ctx context.Context, in *banktypes.QueryAllBalancesRequest, opts ...grpc.CallOption) (*banktypes.QueryAllBalancesResponse, error)
}

// ParamsCache is an interface for a simple in-memory cache implementation for onchain module parameter quueries.
// It does not involve key-value pairs, but only stores a single value.
type ParamsCache[T any] interface {
	Get() (T, bool)
	Set(T)
	Clear()
}
