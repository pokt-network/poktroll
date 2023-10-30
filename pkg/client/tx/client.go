package tx

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/multierr"

	"pocket/pkg/client"
	"pocket/pkg/either"
	"pocket/pkg/observable/channel"
)

const (
	// DefaultCommitTimeoutHeightOffset is the default number of blocks after the
	// latest block (when broadcasting) that a transactions should be considered
	// errored if it has not been committed.
	DefaultCommitTimeoutHeightOffset = 5
	// txWithSenderAddrQueryFmt is the query used to subscribe to cometbft transactions
	// events where the sender address matches the interpolated address.
	// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
	txWithSenderAddrQueryFmt = "tm.event='Tx' AND message.sender='%s'"
)

var _ client.TxClient = (*txClient)(nil)

// txClient is an implementation of the client.TxClient interface responsible for
// transaction operations on Cosmos-based blockchains. It orchestrates building,
// signing, broadcasting, and querying of transactions. The client maintains a
// single events query subscription to its own transactions (via the
// EventsQueryClient) in order to receive notifications regarding their status.
// It also depends on the BlockClient as a timer, synchronized to block height,
// to facilitate transaction timeout logic.
//
// Key Features:
//   - Configurable commit timeout logic: Transactions that are not committed
//     within a specified number of blocks from the latest block are considered
//     errored.
//   - Signing integration: Transactions are signed using a specific key from the
//     keyring identified by its name.
//   - Single event subscription: Subscribes to transaction events that are
//     specifically initiated by the client's address.
//   - Thread safety: Uses a mutex to ensure safe concurrent access to its
//     transaction maps.
//   - Block Monitoring for Transaction Timeout: Through the blockClient, the
//     client monitors committed blocks. If a transaction doesn't appear within
//     the specified number of blocks (controlled by commitTimeoutHeightOffset),
//     it is considered as timed out and an error is delivered.
//
// Internals:
//   - txErrorChans: Maintains a map of transaction hashes to channels which will
//     receive the outcome of the transaction (success or async error).
//   - txTimeoutPool: Used to track transactions that have not yet been committed
//     by a certain block height, enabling the client to implement timeout logic
//     for transactions.
type txClient struct {
	// INCOMPLETE: this should be configurable & integrated w/ viper, flags, etc.
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
	// eventsQueryClient is the client used to subscribe to transactions events from this
	// sender. It is used to receive notifications about transactions events corresponding
	// to transactions which it has constructed, signed, and broadcast.
	eventsQueryClient client.EventsQueryClient
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
}

type (
	txTimeoutPool      map[height]txErrorChansByHash
	txErrorChansByHash map[txHash]chan error
	height             = int64
	txHash             = string
)

// TxEvent is used to deserialize incoming websocket messages from
// the transactions subscription.
type TxEvent struct {
	// Tx is the binary representation of the tx hash.
	Tx     []byte            `json:"tx"`
	Events []abciTypes.Event `json:"events"`
}

// NewTxClient initializes and returns a new instance of TxClient. The function
// sets up the default configurations and applies any provided options to
// customize the client.
//
// Parameters:
// - ctx: The context used for transaction subscriptions.
// - deps: Dependencies injected via depinject.
// - opts: A variadic list of TxClientOption to customize the client's behavior.
//
// The function performs the following steps:
//  1. Initializes a default txClient with the default commit timeout height
//     offset, an empty error channel map, and a transaction timeout pool.
//  2. Injects the necessary dependencies using depinject.
//  3. Applies any provided options to customize the client.
//  4. Validates and sets any missing default configurations using the
//     validateConfigAndSetDefaults method.
//  5. Subscribes the client to its own transactions. This step might be
//     reconsidered for relocation to a potential Start() method in the future.
//
// Returns:
// - An instance of the TxClient if initialization is successful.
// - An error if any of the initialization steps fail.
func NewTxClient(
	ctx context.Context,
	deps depinject.Config,
	opts ...client.TxClientOption,
) (client.TxClient, error) {
	tClient := &txClient{
		commitTimeoutHeightOffset: DefaultCommitTimeoutHeightOffset,
		txErrorChans:              make(txErrorChansByHash),
		txTimeoutPool:             make(txTimeoutPool),
	}

	if err := depinject.Inject(
		deps,
		&tClient.txCtx,
		&tClient.eventsQueryClient,
		&tClient.blockClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(tClient)
	}

	if err := tClient.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	// TODO_CONSIDERATION: move this into a #Start() method
	if err := tClient.subscribeToOwnTxs(ctx); err != nil {
		return nil, err
	}
	return tClient, nil
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
// If any step encounters an error, it returns an AsyncError populated with the
// synchronous error. If the function completes successfully, it returns an
// AsyncError populated with the result of the addPendingTransactions call.
//
// Parameters:
// - ctx: The context used for this operation.
// - msgs: A variadic list of Cosmos SDK messages to be signed and broadcast.
//
// Returns:
//   - An AsyncError which can be either a synchronous error or the result of
//     the addPendingTransactions call.
func (tClient *txClient) SignAndBroadcast(
	ctx context.Context,
	msgs ...cosmostypes.Msg,
) either.AsyncError {
	var validationErrs error
	for i, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			validationErr := ErrInvalidMsg.Wrapf("in msg with index %d: %s", i, err)
			validationErrs = multierr.Append(validationErrs, validationErr)
		}
	}
	if validationErrs != nil {
		return either.SyncErr(validationErrs)
	}

	// Construct the transactions using cosmos' transactions builder.
	txBuilder := tClient.txCtx.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		// return synchronous error
		return either.SyncErr(err)
	}

	// Calculate timeout height
	timeoutHeight := tClient.blockClient.LatestBlock(ctx).
		Height() + tClient.commitTimeoutHeightOffset

	// TODO_TECHDEBT: this should be configurable
	txBuilder.SetGasLimit(200000)
	txBuilder.SetTimeoutHeight(uint64(timeoutHeight))

	// sign transactions
	err := tClient.txCtx.SignTx(
		tClient.signingKeyName,
		txBuilder,
		false, false,
	)
	if err != nil {
		return either.SyncErr(err)
	}

	// ensure transactions is valid
	// NOTE: this makes the transactions valid; i.e. it is *REQUIRED*
	if err := txBuilder.GetTx().ValidateBasic(); err != nil {
		return either.SyncErr(err)
	}

	// serialize transactions
	txBz, err := tClient.txCtx.EncodeTx(txBuilder)
	if err != nil {
		return either.SyncErr(err)
	}

	txResponse, err := tClient.txCtx.BroadcastTx(txBz)
	if err != nil {
		return either.SyncErr(err)
	}

	if txResponse.Code != 0 {
		return either.SyncErr(ErrCheckTx.Wrapf(txResponse.RawLog))
	}

	return tClient.addPendingTransactions(normalizeTxHashHex(txResponse.TxHash), timeoutHeight)
}

// validateConfigAndSetDefaults ensures that the necessary configurations for the
// txClient are set, and populates any missing defaults.
//
//  1. It checks if the signing key name is set and returns an error if it's empty.
//  2. It then retrieves the key record from the keyring using the signing key name
//     and checks its existence.
//  3. The signing address for the key record is determined and set to the txClient.
//  4. Lastly, it ensures that commitTimeoutHeightOffset has a valid value, setting
//     it to DefaultCommitTimeoutHeightOffset if it's zero or negative.
//
// Returns:
// - ErrEmptySigningKeyName if the signing key name is not provided.
// - ErrNoSuchSigningKey if the signing key is not found in the keyring.
// - ErrSigningKeyAddr if there's an issue retrieving the address for the signing key.
// - nil if validation is successful and defaults are set appropriately.
func (tClient *txClient) validateConfigAndSetDefaults() error {
	if tClient.signingKeyName == "" {
		return ErrEmptySigningKeyName
	}

	keyRecord, err := tClient.txCtx.GetKeyring().Key(tClient.signingKeyName)
	if err != nil {
		return ErrNoSuchSigningKey.Wrapf("name %q: %s", tClient.signingKeyName, err)
	}
	signingAddr, err := keyRecord.GetAddress()
	if err != nil {
		return ErrSigningKeyAddr.Wrapf("name %q: %s", tClient.signingKeyName, err)
	}
	tClient.signingAddr = signingAddr

	if tClient.commitTimeoutHeightOffset <= 0 {
		tClient.commitTimeoutHeightOffset = DefaultCommitTimeoutHeightOffset
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
// channel for a given transaction hash. This design allows for efficient error handling
// both in terms of transaction-specific errors and transaction timeout logic.
//
// Note: The error channels are buffered to prevent blocking on send operations and
// are intended to convey a single error event.
//
// Parameters:
// - txHash: The unique hash of the pending transaction.
// - timeoutHeight: The block height at which the transaction is considered timed out.
//
// Returns:
//   - An AsyncError containing a reference to the error notification channel for the
//     provided transaction hash.
func (tClient *txClient) addPendingTransactions(
	txHash string,
	timeoutHeight int64,
) either.AsyncError {
	tClient.txsMutex.Lock()
	defer tClient.txsMutex.Unlock()

	// Initialize txTimeoutPool map if necessary.
	txsByHash, ok := tClient.txTimeoutPool[timeoutHeight]
	if !ok {
		txsByHash = make(map[string]chan error)
		tClient.txTimeoutPool[timeoutHeight] = txsByHash
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
	if _, ok := tClient.txErrorChans[txHash]; !ok {
		// NB: both maps hold a reference to the same channel so that we can check
		// if the channel has already been closed when timing out.
		tClient.txErrorChans[txHash] = errCh
	}

	return either.AsyncErr(errCh)
}

// subscribeToOwnTxs establishes an event query subscription to monitor transactions
// originating from this client's signing address. It efficiently listens for and
// handles transaction events to facilitate real-time feedback and error management.
//
// The function performs the following primary operations:
//
//  1. Forms a query to fetch transaction events specific to the client's signing address.
//  2. Transforms raw events into a stream of transaction events using the txEventFromEventBz function.
//  3. Processes each transaction event:
//     a. Converts the transaction hash to its normalized hexadecimal representation.
//     b. Retrieves and manages the associated error notification channel from txErrorChans.
//     c. Closes and removes the notification channel after processing.
//     d. Cleans up the txTimeoutPool, removing transaction entries that are no longer relevant.
//
// Important considerations:
// There's uncertainty surrounding the potential for asynchronous errors post transaction broadcast.
// Current implementation and observations suggest that errors might be returned synchronously,
// even when using Cosmos' BroadcastTxAsync method. Further investigation is required.
//
// This function also spawns a goroutine to handle transaction timeouts via goTimeoutPendingTransactions.
//
// Parameters:
// - ctx: Context for managing the function's lifecycle and child operations.
//
// Returns:
// - An error if there's a failure during the event query or subscription process.
func (tClient *txClient) subscribeToOwnTxs(ctx context.Context) error {
	// Form a query based on the client's signing address.
	query := fmt.Sprintf(txWithSenderAddrQueryFmt, tClient.signingAddr)

	// Fetch transaction events matching the query.
	eventsBz, err := tClient.eventsQueryClient.EventsBytes(ctx, query)
	if err != nil {
		return err
	}

	// Convert raw event data into a stream of transaction events.
	txEventsObservable := channel.Map[
		either.Bytes, either.Either[*TxEvent],
	](ctx, eventsBz, tClient.txEventFromEventBz)
	txEventsObserver := txEventsObservable.Subscribe(ctx)

	// Goroutine to handle each incoming transaction event.
	go func() {
		for eitherTxEvent := range txEventsObserver.Ch() {
			txEvent, err := eitherTxEvent.ValueOrError()
			if err != nil {
				return
			}

			// Convert transaction hash into its normalized hex form.
			txHashHex := txHashBytesToNormalizedHex(comettypes.Tx(txEvent.Tx).Hash())

			tClient.txsMutex.Lock()

			// Check for a corresponding error channel in the map.
			txErrCh, ok := tClient.txErrorChans[txHashHex]
			if !ok {
				panic("Received tx event without an associated error channel.")
			}

			// TODO_INVESTIGATE: it seems like it may not be possible for the
			// txEvent to represent an error. Cosmos' #BroadcastTxSync() is being
			// called internally, which will return an error if the transaction
			// is not accepted by the mempool.
			//
			// It's unclear if a cosmos chain is capable of returning an async
			// error for a transaction at this point; even when substituting
			// #BroadcastTxAsync(), the error is returned synchronously:
			//
			// > error in json rpc client, with http response metadata: (Status:
			// > 200 OK, Protocol HTTP/1.1). RPC error -32000 - tx added to local
			// > mempool but failed to gossip: validation failed
			//
			// Potential parse and send transaction error on txErrCh here.

			close(txErrCh)
			delete(tClient.txErrorChans, txHashHex)

			// Clean up error channels in the timeout pool post processing.
			for timeoutHeight, txErrorChans := range tClient.txTimeoutPool {
				for txHash := range txErrorChans {
					if txHash == txHash {
						delete(txErrorChans, txHash)
					}
				}
				if len(txErrorChans) == 0 {
					delete(tClient.txTimeoutPool, timeoutHeight)
				}
			}

			tClient.txsMutex.Unlock()
		}
	}()

	// Launch a separate goroutine to handle transaction timeouts.
	go tClient.goTimeoutPendingTransactions(ctx)

	return nil
}

// goTimeoutPendingTransactions monitors blocks and handles transaction timeouts.
// For each block observed, it checks if there are transactions associated with that
// block's height in the txTimeoutPool. If transactions are found, the function
// evaluates whether they have been processed by the subscription:
//   - If processed, the error channel associated with the transaction (txErrCh) should
//     be closed by the websocket message handler.
//   - If not processed, a timeout error is sent on the error channel, and it is then
//     closed and removed.
//
// The function performs efficient cleanup operations, removing processed transactions
// and ensuring the txTimeoutPool remains up-to-date.
//
// Important:
// This function is designed to run as a dedicated goroutine, continuously monitoring
// for new blocks and managing transaction timeouts until the provided context is done.
//
// Parameters:
// - ctx: Context for managing the function's lifecycle and child operations.
// goTimeoutPendingTransactions monitors incoming blocks to manage the timeout of
// pending transactions. If a transaction associated with a block's height hasn't
// been processed by its associated subscription, a timeout error is sent through its
// respective error channel. This function is designed to run as a dedicated
// goroutine.
//
// Parameters:
// - ctx: Context for managing the function's lifecycle and child operations.
func (tClient *txClient) goTimeoutPendingTransactions(ctx context.Context) {
	// Subscribe to a sequence of committed blocks.
	blockCh := tClient.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx).Ch()

	// Iterate over each incoming block.
	for block := range blockCh {
		select {
		case <-ctx.Done():
			// Exit if the context signals done.
			return
		default:
		}

		tClient.txsMutex.Lock()

		// Retrieve transactions associated with the current block's height.
		txsByHash, ok := tClient.txTimeoutPool[block.Height()]
		if !ok {
			// If no transactions are found for the current block height, continue.
			tClient.txsMutex.Unlock()
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
				tClient.txsMutex.Unlock()
				continue
			default:
			}

			// Transaction was not processed by its subscription: handle timeout.
			txErrCh <- tClient.getTxTimeoutError(ctx, txHash) // Send a timeout error.
			close(txErrCh)                                    // Close the error channel.
			delete(txsByHash, txHash)                         // Remove the transaction.
		}

		// Clean up the txTimeoutPool for the current block height.
		delete(tClient.txTimeoutPool, block.Height())
		tClient.txsMutex.Unlock()
	}
}

// txEventFromEventBz deserializes a binary representation of a transaction event
// into a TxEvent structure. This function is primarily designed as a websocket
// message handler to transform raw byte data into structured transaction events.
// It also manages error channels (txErrCh), ensuring their proper closure and
// sending errors when applicable.
//
// The function proceeds in the following manner:
// 1. Extracts valid byte data, returning an error if extraction fails.
// 2. Unmarshals the byte data into a TxEvent structure.
// 3. Manages the error channels contingent on the result of unmarshalling.
//
// Parameters:
//   - eitherEventBz: Binary data of the event, potentially encapsulating an error.
//
// Returns:
//   - eitherTxEvent: The TxEvent or an encapsulated error, facilitating clear
//     error management in the caller's context.
//   - skip: A flag denoting if the event should be bypassed. A value of true
//     suggests the event be disregarded, progressing to the succeeding message.
func (tClient *txClient) txEventFromEventBz(
	eitherEventBz either.Bytes,
) (eitherTxEvent either.Either[*TxEvent], skip bool) {

	// Extract byte data from the given event. In case of failure, wrap the error
	// and denote the event for skipping.
	eventBz, err := eitherEventBz.ValueOrError()
	if err != nil {
		return either.Error[*TxEvent](err), true
	}

	// Unmarshal byte data into a TxEvent object.
	txEvt, err := tClient.unmarshalTxEvent(eventBz)
	switch {
	// If the error indicates a non-transactional event, return the TxEvent and
	// signal for skipping.
	case errors.Is(err, ErrNonTxEventBytes):
		return either.Success(txEvt), true
	// For other errors, wrap them and flag the event to be skipped.
	case err != nil:
		return either.Error[*TxEvent](ErrUnmarshalTx.Wrapf("%s", err)), true
	}

	// For successful unmarshalling, return the TxEvent.
	return either.Success(txEvt), false
}

// unmarshalTxEvent attempts to deserialize a slice of bytes into a TxEvent.
// It checks if the given bytes correspond to a valid transaction event.
// If the resulting TxEvent has empty transaction bytes, it assumes that
// the message was not a transaction event and returns an ErrNonTxEventBytes error.
func (tClient *txClient) unmarshalTxEvent(eventBz []byte) (*TxEvent, error) {
	txEvent := new(TxEvent)

	// Try to deserialize the provided bytes into a TxEvent.
	if err := json.Unmarshal(eventBz, txEvent); err != nil {
		return nil, err
	}

	// Check if the TxEvent has empty transaction bytes, which indicates
	// the message might not be a valid transaction event.
	if bytes.Equal(txEvent.Tx, []byte{}) {
		return nil, ErrNonTxEventBytes.Wrapf("%s", string(eventBz))
	}

	return txEvent, nil
}

// getTxTimeoutError checks if a transaction with the specified hash has timed out.
// The function decodes the provided hexadecimal hash into bytes and queries the
// transaction using the byte hash. If any error occurs during this process,
// appropriate wrapped errors are returned for easier debugging.
func (tClient *txClient) getTxTimeoutError(ctx context.Context, txHashHex string) error {

	// Decode the provided hex hash into bytes.
	txHash, err := hex.DecodeString(txHashHex)
	if err != nil {
		return ErrInvalidTxHash.Wrapf("%s", txHashHex)
	}

	// Query the transaction using the decoded byte hash.
	txResponse, err := tClient.txCtx.QueryTx(ctx, txHash, false)
	if err != nil {
		return ErrQueryTx.Wrapf("with hash: %s: %s", txHashHex, err)
	}

	// Return a timeout error with details about the transaction.
	return ErrTxTimeout.Wrapf("with hash %s: %s", txHashHex, txResponse.TxResult.Log)
}
