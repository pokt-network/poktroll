package tx

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/multierr"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/encoding"
)

const (
	// DefaultCommitTimeoutHeightOffset is the default number of blocks after the
	// latest block (when broadcasting) that a transactions should be considered
	// errored if it has not been committed.
	// TODO_TECHDEBT: populate this from the config file.
	DefaultCommitTimeoutHeightOffset = 5

	// defaultTxReplayLimit is the number of comettypes.EventDataTx events that the replay
	// observable returned by LastNBlocks() will be able to replay.
	// TODO_TECHDEBT/TODO_FUTURE: add a `blocksReplayLimit` field to the blockClient
	// struct that defaults to this but can be overridden via an option.
	defaultTxReplayLimit = 100

	// txWithSenderAddrQueryFmt is the query used to subscribe to cometbft transactions
	// events where the sender address matches the interpolated address.
	// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
	txWithSenderAddrQueryFmt = "tm.event='Tx' AND message.sender='%s'"
)

// TODO_TECHDEBT(@bryanchriswhite): Refactor this to use the EventsReplayClient
// In order to simplify the logic of the TxClient
var _ client.TxClient = (*txClient)(nil)

// CometTxEvent is used to deserialize incoming transaction event messages
// from the respective events query subscription. This structure is adapted
// to handle CometBFT's unique serialization format, which diverges from
// conventional approaches seen in implementations like rollkit's. The design
// ensures accurate parsing and compatibility with CometBFT's serialization
// of transaction results.
type CometTxEvent struct {
	Data struct {
		// TxResult is nested to accommodate CometBFT's serialization format,
		// ensuring correct deserialization of transaction results.
		Value struct {
			TxResult abci.TxResult
		} `json:"value"`
	} `json:"data"`
}

// txClient orchestrates building, signing, broadcasting, and querying of
// transactions. It maintains a single events query subscription to its own
// transactions (via the EventsQueryClient) in order to receive notifications
// regarding their status.
// It also depends on the BlockClient as a timer, synchronized to block height,
// to facilitate transaction timeout logic. If a transaction doesn't appear to
// have been committed by commitTimeoutHeightOffset number of blocks have elapsed,
// it is considered as timed out. Upon timeout, the client queries the network for
// the last status of the transaction, which is used to derive the asynchronous
// error that's populated in the either.AsyncError.
type txClient struct {
	// TODO_TECHDEBT: this should be configurable & integrated w/ viper, flags, etc.
	// commitTimeoutHeightOffset is the number of blocks after the latest block
	// that a transactions should be considered errored if it has not been committed.
	commitTimeoutHeightOffset int64
	// signingKeyName is the name of the key in the keyring to use for signing
	// transactions.
	signingKeyName string
	// signingAddr is the address of the signing key referenced by signingKeyName.
	// It is hydrated from the keyring by calling Keyring#Key() with signingKeyName.
	signingAddr cosmostypes.AccAddress
	// txCtx is the transactions context which encapsulates transactions building, signing,
	// broadcasting, and querying, as well as keyring access.
	txCtx client.TxContext
	// eventsReplayClient is the client used to subscribe to transactions events from this
	// sender. It is used to receive notifications about transactions events corresponding
	// to transactions which it has constructed, signed, and broadcast.
	eventsReplayClient client.EventsReplayClient[*abci.TxResult]
	// blockClient is the client used to query for the latest block height.
	// It is used to implement timout logic for transactions which weren't committed.
	blockClient client.BlockClient

	// txsMutex protects txErrorChans and txTimeoutPool maps.
	txsMutex sync.Mutex
	// txErrorChans maps tx_hash->channel which will receive an error or nil,
	// and close, when the transactions with the given hash is committed.
	txErrorChans txErrorChansByHash
	// txTimeoutPool maps timeout_block_height->map_of_txsByHash. It
	// is used to ensure that transactions error channels receive and close in the event
	// that they have not already by the given timeout height.
	txTimeoutPool txTimeoutPool

	// gasPrices is the gas unit prices used for sending transactions.
	gasPrices cosmostypes.DecCoins

	// connRetryLimit is the number of times the underlying replay client
	// should retry in the event that it encounters an error or its connection is interrupted.
	// If connRetryLimit is < 0, it will retry indefinitely.
	connRetryLimit int
}

type (
	txTimeoutPool      map[height]txErrorChansByHash
	txErrorChansByHash map[txHash]chan error
	height             = int64
	txHash             = string
)

// NewTxClient attempts to construct a new TxClient using the given dependencies
// and options.
//
// It performs the following steps:
//  1. Initializes a default txClient with the default commit timeout height
//     offset, an empty error channel map, and an empty transaction timeout pool.
//  2. Injects the necessary dependencies using depinject.
//  3. Applies any provided options to customize the client.
//  4. Validates and sets any missing default configurations using the
//     validateConfigAndSetDefaults method.
//  5. Subscribes the client to its own transactions. This step might be
//     reconsidered for relocation to a potential Start() method in the future.
//
// Required dependencies:
//   - client.TxContext
//   - client.EventsQueryClient
//   - client.BlockClient
//
// Available options:
//   - WithSigningKeyName
//   - WithCommitTimeoutHeightOffset
func NewTxClient(
	ctx context.Context,
	deps depinject.Config,
	opts ...client.TxClientOption,
) (_ client.TxClient, err error) {
	txnClient := &txClient{
		commitTimeoutHeightOffset: DefaultCommitTimeoutHeightOffset,
		txErrorChans:              make(txErrorChansByHash),
		txTimeoutPool:             make(txTimeoutPool),
	}

	if err = depinject.Inject(
		deps,
		&txnClient.txCtx,
		&txnClient.blockClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(txnClient)
	}

	if err = txnClient.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	// Form a query based on the client's signing address.
	eventQuery := fmt.Sprintf(txWithSenderAddrQueryFmt, txnClient.signingAddr)

	// Initialize and events replay client.
	txnClient.eventsReplayClient, err = events.NewEventsReplayClient[*abci.TxResult](
		ctx,
		deps,
		eventQuery,
		UnmarshalTxResult,
		defaultTxReplayLimit,
		events.WithConnRetryLimit[*abci.TxResult](txnClient.connRetryLimit),
	)
	if err != nil {
		return nil, err
	}

	// Start an events query subscription for transactions originating from this
	// client's signing address.
	// TODO_CONSIDERATION: move this into a #Start() method
	go txnClient.goSubscribeToOwnTxs(ctx)

	// Launch a separate goroutine to handle transaction timeouts.
	// TODO_CONSIDERATION: move this into a #Start() method
	go txnClient.goTimeoutPendingTransactions(ctx)

	return txnClient, nil
}

// SignAndBroadcast signs a set of Cosmos SDK messages, constructs a transaction,
// and broadcasts it to the network. The function performs several steps to
// ensure the messages and the resultant transaction are valid:
//
//  1. Validates each message in the provided set.
//  2. Constructs the transaction using the Cosmos SDK's transaction builder.
//  3. Calculates and sets the transaction's timeout height.
//  4. Sets a default gas limit (note: this will be made configurable in the future).
//  5. Signs the transaction.
//  6. Validates the constructed transaction.
//  7. Serializes and broadcasts the transaction.
//  8. Checks the broadcast response for errors.
//  9. If all the above steps are successful, the function registers the
//     transaction as pending.
//
// If any step encounters an error, it returns an either.AsyncError populated with
// the synchronous error. If the function completes successfully, it returns an
// either.AsyncError populated with the error channel which will receive if the
// transaction results in an asynchronous error or times out.
func (txnClient *txClient) SignAndBroadcast(
	ctx context.Context,
	msgs ...cosmostypes.Msg,
) either.AsyncError {
	var validationErrs error
	for i, msg := range msgs {
		validatableMsg, ok := msg.(cosmostypes.HasValidateBasic)
		if ok {
			if err := validatableMsg.ValidateBasic(); err != nil {
				validationErr := ErrInvalidMsg.Wrapf("in msg with index %d: %s", i, err)
				validationErrs = multierr.Append(validationErrs, validationErr)
			}
		}
	}
	if validationErrs != nil {
		return either.SyncErr(validationErrs)
	}

	// Simulate the transaction to calculate the gas limit.
	gasLimit, simErr := txnClient.txCtx.GetSimulatedTxGas(ctx, txnClient.signingKeyName, msgs...)
	if simErr != nil {
		return either.SyncErr(simErr)
	}

	// Construct the transactions using cosmos' transactions builder.
	txBuilder := txnClient.txCtx.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		// return synchronous error
		return either.SyncErr(err)
	}

	// Calculate timeout height
	timeoutHeight := txnClient.blockClient.LastBlock(ctx).
		Height() + txnClient.commitTimeoutHeightOffset

	txBuilder.SetGasLimit(gasLimit)

	gasLimitDec := math.LegacyNewDec(int64(gasLimit))
	feeAmountDec := txnClient.gasPrices.MulDec(gasLimitDec)

	feeCoins, changeCoins := feeAmountDec.TruncateDecimal()
	// Ensure that any decimal remainder is added to the corresponding coin as a
	// whole number.
	// Since changeCoins is the result of DecCoins#TruncateDecimal, it will always
	// be less than 1 unit of the feeCoins.
	if !changeCoins.IsZero() {
		feeCoins = feeCoins.Add(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))
	}
	txBuilder.SetFeeAmount(feeCoins)

	txBuilder.SetTimeoutHeight(uint64(timeoutHeight))

	// sign transactions
	err := txnClient.txCtx.SignTx(
		txnClient.signingKeyName,
		txBuilder,
		false, false,
	)
	if err != nil {
		return either.SyncErr(err)
	}

	// ensure transactions is valid
	// NOTE: this makes the transactions valid; i.e. it is *REQUIRED*
	if err = txBuilder.GetTx().ValidateBasic(); err != nil {
		return either.SyncErr(err)
	}

	// serialize transactions
	txBz, err := txnClient.txCtx.EncodeTx(txBuilder)
	if err != nil {
		return either.SyncErr(err)
	}

	txResponse, err := txnClient.txCtx.BroadcastTx(txBz)
	if err != nil {
		return either.SyncErr(err)
	}

	if txResponse.Code != 0 {
		return either.SyncErr(ErrCheckTx.Wrapf("%s", txResponse.RawLog))
	}

	return txnClient.addPendingTransactions(encoding.NormalizeTxHashHex(txResponse.TxHash), timeoutHeight)
}

// validateConfigAndSetDefaults ensures that the necessary configurations for the
// txClient are set, and populates any missing defaults.
//
//  1. It checks if the signing key name is set and returns an error if it's empty.
//  2. It then retrieves the key record from the keyring using the signing key name
//     and checks its existence.
//  3. The address of the signing key is computed and assigned to txClient#signgingAddr.
//  4. Lastly, it ensures that commitTimeoutHeightOffset has a valid value, setting
//     it to DefaultCommitTimeoutHeightOffset if it's zero or negative.
//
// Returns:
// - ErrEmptySigningKeyName if the signing key name is not provided.
// - ErrNoSuchSigningKey if the signing key is not found in the keyring.
// - ErrSigningKeyAddr if there's an issue retrieving the address for the signing key.
// - nil if validation is successful and defaults are set appropriately.
func (txnClient *txClient) validateConfigAndSetDefaults() error {
	signingAddr, err := keyring.KeyNameToAddr(
		txnClient.signingKeyName,
		txnClient.txCtx.GetKeyring(),
	)
	if err != nil {
		return err
	}

	txnClient.signingAddr = signingAddr

	if txnClient.commitTimeoutHeightOffset <= 0 {
		txnClient.commitTimeoutHeightOffset = DefaultCommitTimeoutHeightOffset
	}
	return nil
}

// addPendingTransactions registers a new pending transaction for monitoring and
// notification of asynchronous errors. It accomplishes the following:
//
//  1. Creates an error notification channel (if one doesn't already exist) and associates
//     it with the provided transaction hash in the txErrorChans map.
//
//  2. Ensures that there's an initialized map of transactions by hash for the
//     given timeout height in the txTimeoutPool. The same error notification channel
//     is also associated with the transaction hash in this map.
//
// Both txErrorChans and txTimeoutPool store references to the same error notification
// channel for a given transaction hash. This ensures idempotency of error handling
// for any given transaction between asynchronous, transaction-specific errors and
// transaction timeout logic.
//
// Note: The error channels are buffered to prevent blocking on send operations and
// are intended to convey a single error event.
//
// Returns:
//   - An either.AsyncError populated with the error notification channel for the
//     provided transaction hash.
func (txnClient *txClient) addPendingTransactions(
	txHash string,
	timeoutHeight int64,
) either.AsyncError {
	txnClient.txsMutex.Lock()
	defer txnClient.txsMutex.Unlock()

	// Initialize txTimeoutPool map if necessary.
	txsByHash, ok := txnClient.txTimeoutPool[timeoutHeight]
	if !ok {
		txsByHash = make(map[string]chan error)
		txnClient.txTimeoutPool[timeoutHeight] = txsByHash
	}

	// Initialize txErrorChans map in txTimeoutPool map if necessary.
	errCh, ok := txsByHash[txHash]
	if !ok {
		// NB: intentionally buffered to avoid blocking on send. Only intended
		// to send/receive a single error.
		errCh = make(chan error, 1)
		txsByHash[txHash] = errCh
	}

	// Initialize txErrorChans map if necessary.
	if _, ok := txnClient.txErrorChans[txHash]; !ok {
		// NB: both maps hold a reference to the same channel so that we can check
		// if the channel has already been closed when timing out.
		txnClient.txErrorChans[txHash] = errCh
	}

	return either.AsyncErr(errCh)
}

// goSubscribeToOwnTxs establishes an event query subscription to monitor transactions
// originating from this client's signing address, ranges over observed transaction events,
// and performs the following steps on each:
//
//  1. Normalize hexadeimal transaction hash.
//  2. Retrieves the transaction's error channel from txErrorChans.
//  3. Closes and removes it from txErrorChans.
//  4. Removes the transaction error channel from txTimeoutPool.
//
// It is intended to be called in a goroutine.
//
// Important considerations:
// There's uncertainty surrounding the potential for asynchronous errors post transaction broadcast.
// Current implementation and observations suggest that errors might be returned synchronously,
// even when using Cosmos' BroadcastTxAsync method. Further investigation is required.
//
// Parameters:
// - ctx: Context for managing the function's lifecycle and child operations.
func (txnClient *txClient) goSubscribeToOwnTxs(ctx context.Context) {
	// Create a cancellable child context for managing the EventsSequence lifecycle.
	// Canceling it when listening for submitted transactions are no longer needed
	// will ensure that the EventsSequence, its subscriptions and underlying channels
	// are effectively closed.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	txResultsObs := txnClient.eventsReplayClient.EventsSequence(ctx)
	txResultsCh := txResultsObs.Subscribe(ctx).Ch()
	for txResult := range txResultsCh {
		// Convert transaction hash into its normalized hex form.
		txHashHex := encoding.TxHashBytesToNormalizedHex(comettypes.Tx(txResult.Tx).Hash())

		txnClient.txsMutex.Lock()
		// Remove from the txTimeoutPool.
		for timeoutHeight, txErrorChans := range txnClient.txTimeoutPool {
			// Handled transaction isn't in this timeout height or is an external transaction.
			if _, ok := txErrorChans[txHashHex]; !ok {
				continue
			}

			delete(txErrorChans, txHashHex)
			if len(txErrorChans) == 0 {
				delete(txnClient.txTimeoutPool, timeoutHeight)
			}

			// Transactions that are initiated externally will not have an associated
			// error channel in txClient.txErrorChans and are already skipped by
			// the "continue" statement above.
			txErrCh, ok := txnClient.txErrorChans[txHashHex]
			if !ok {
				panic("Received tx event without an associated error channel.")
			}

			// TODO_INVESTIGATE: It seems that txResult does not represent transactions
			// that are not accepted by the mempool and #BroadcastTxAsync() synchronously
			// returns the following error:
			//
			// > error in json rpc client, with http response metadata: (Status:
			// > 200 OK, Protocol HTTP/1.1). RPC error -32000 - tx added to local
			// > mempool but failed to gossip: validation failed
			//
			// Potential parse and send transaction error on txErrCh here.

			// Close and remove from txErrChans
			close(txErrCh)
			delete(txnClient.txErrorChans, txHashHex)
		}

		txnClient.txsMutex.Unlock()
	}
}

// goTimeoutPendingTransactions monitors blocks and handles transaction timeouts.
// For each block observed, it checks if there are transactions associated with that
// block's height in the txTimeoutPool. If transactions are found, the function
// evaluates whether they have already been processed by the transaction events
// query subscription logic. If not, a timeout error is generated and sent on the
// transaction's error channel. Finally, the error channel is closed and removed
// from the txTimeoutPool.
func (txnClient *txClient) goTimeoutPendingTransactions(ctx context.Context) {
	// Subscribe to a sequence of committed blocks.
	blockCh := txnClient.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx).Ch()

	// Iterate over each incoming block.
	for block := range blockCh {
		select {
		case <-ctx.Done():
			// Exit if the context signals done.
			return
		default:
		}

		txnClient.txsMutex.Lock()

		// Retrieve transactions associated with the current block's height.
		txsByHash, ok := txnClient.txTimeoutPool[block.Height()]
		if !ok {
			// If no transactions are found for the current block height, continue.
			txnClient.txsMutex.Unlock()
			continue
		}

		// Process each transaction for the current block height.
		for txHash, txErrCh := range txsByHash {
			select {
			// Check if the transaction was processed by its subscription.
			case err, ok := <-txErrCh:
				if ok {
					// Unexpected state: error channel should be closed after processing.
					panic(fmt.Errorf("Expected txErrCh to be closed; received err: %w", err))
				}
				// Remove the processed transaction.
				delete(txsByHash, txHash)
				txnClient.txsMutex.Unlock()
				continue
			default:
			}

			// Transaction was not processed by its subscription: handle timeout.
			txErrCh <- txnClient.getTxTimeoutError(ctx, txHash) // Send a timeout error.
			close(txErrCh)                                      // Close the error channel.
			delete(txsByHash, txHash)                           // Remove the transaction.
		}

		// Clean up the txTimeoutPool for the current block height.
		delete(txnClient.txTimeoutPool, block.Height())
		txnClient.txsMutex.Unlock()
	}
}

// getTxTimeoutError checks if a transaction with the specified hash has timed out.
// The function decodes the provided hexadecimal hash into bytes and queries the
// transaction using the byte hash. If any error occurs during this process,
// appropriate wrapped errors are returned for easier debugging.
func (txnClient *txClient) getTxTimeoutError(ctx context.Context, txHashHex string) error {
	// Decode the provided hex hash into bytes.
	txHash, err := hex.DecodeString(txHashHex)
	if err != nil {
		return ErrInvalidTxHash.Wrapf("%s", txHashHex)
	}

	// Query the transaction using the decoded byte hash.
	txResponse, err := txnClient.txCtx.QueryTx(ctx, txHash, false)
	if err != nil {
		return ErrQueryTx.Wrapf("with hash: %s: %s", txHashHex, err)
	}

	// Return a timeout error with details about the transaction.
	return ErrTxTimeout.Wrapf("with hash %s: %s", txHashHex, txResponse.TxResult.Log)
}

// UnmarshalTxResult attempts to deserialize a slice of bytes into a TxResult
// It checks if the given bytes correspond to a valid transaction event.
// If the resulting TxResult has empty transaction bytes, it assumes that
// the message was not a transaction results and returns an error.
func UnmarshalTxResult(txResultBz []byte) (*abci.TxResult, error) {
	var rpcResponse rpctypes.RPCResponse

	// Try to deserialize the provided bytes into an RPCResponse.
	if err := json.Unmarshal(txResultBz, &rpcResponse); err != nil {
		return nil, events.ErrEventsUnmarshalEvent.Wrap(err.Error())
	}

	var cometTxEvent CometTxEvent
	// Try to deserialize the provided bytes into a CometTxEvent.
	if err := json.Unmarshal(rpcResponse.Result, &cometTxEvent); err != nil {
		return nil, events.ErrEventsUnmarshalEvent.Wrap(err.Error())
	}

	// Check if the TxResult has empty transaction bytes, which indicates
	// the message might not be a valid transaction event.
	if bytes.Equal(cometTxEvent.Data.Value.TxResult.Tx, []byte{}) {
		return nil, events.ErrEventsUnmarshalEvent.Wrap("event bytes do not correspond to an abci.TxResult")
	}

	return &cometTxEvent.Data.Value.TxResult, nil
}
