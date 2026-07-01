package tx

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/multierr"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/encoding"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
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

	// MUST set a timeout when using unordered transactions. 10 minutes is the maximum, use 9 for safety.
	// See: https://docs.cosmos.network/v0.53/build/architecture/adr-070-unordered-account
	txTimeoutTimestampDelay = time.Minute * 9

	// maxTxRebroadcasts is the number of times the tx client will re-broadcast a
	// still-pending (un-included) transaction before its timeout height. A
	// claim/proof tx is broadcast exactly once at its assigned commit height; if
	// that broadcast is evicted from the mempool before inclusion (window-open
	// burst exceeding the block gas limit, a mempool recheck eviction, a node
	// restart, ...) it is never re-injected and the claim/proof is forfeited at
	// window close (PROOF_MISSING) even when later blocks in the window are empty.
	// Re-broadcasts are spread evenly across the window (see collectDueRebroadcasts)
	// so a single bad block does not sink the tx. Two recovers the residual tail
	// observed on mainnet where a single mid-window resend still missed (the resend
	// itself landing in a second congested/empty block). Keep small to avoid
	// flooding the mempool with duplicates.
	// TODO_TECHDEBT: make this (and txRebroadcastSafetyBlocks) configurable.
	maxTxRebroadcasts = 2

	// txRebroadcastSafetyBlocks stops re-broadcasting once the chain is within this
	// many blocks of a tx's timeout height, since a resend that late cannot land.
	txRebroadcastSafetyBlocks = 1
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

// txClient orchestrates building, signing, broadcasting, and querying of transactions.
//
// It maintains a single events query subscription to its own transactions (via the
// EventsQueryClient) to receive status notifications.
//
// Dependencies:
// - Uses BlockClient as a synchronized block-height timer for transaction timeout logic
// - If a transaction isn't committed by timeoutHeight, it's considered timed out
//
// Timeout handling:
// - Upon timeout, the client queries the network for the transaction's last status
// - This status is used to derive the asynchronous error populated in either.AsyncError
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
	// It is used to implement timeout logic for transactions which weren't committed.
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
	// rebroadcastPool maps tx_hash->pendingRebroadcast for transactions that have
	// been broadcast but not yet observed on-chain. The committed-block loop uses
	// it to re-broadcast txs that may have been evicted from the mempool before
	// inclusion (see rebroadcastDuePendingTxs). Guarded by txsMutex.
	rebroadcastPool map[txHash]*pendingRebroadcast

	// gasPrices is the gas unit prices used for sending transactions.
	gasPrices *cosmostypes.DecCoins

	// gasAdjustment is the gas adjustment factor used for sending transactions.
	gasAdjustment float64

	// gasSetting is the gas setting used for sending transactions.
	gasSetting *flags.GasSetting

	// feeAmount is the fee amount used for sending transactions.
	feeAmount *cosmostypes.DecCoins

	// connRetryLimit is the number of times the underlying replay client
	// should retry in the event that it encounters an error or its connection is interrupted.
	// If connRetryLimit is < 0, it will retry indefinitely.
	connRetryLimit int

	// unordered is a flag which indicates whether the transactions should be sent unordered.
	unordered bool

	logger polylog.Logger
}

type (
	txTimeoutPool      map[height]txErrorChansByHash
	txErrorChansByHash map[txHash]chan error
	height             = int64
	txHash             = string
)

// pendingRebroadcast holds the signed bytes and scheduling metadata required to
// re-broadcast a transaction that has not yet been observed on-chain.
type pendingRebroadcast struct {
	// txBz is the signed transaction bytes. Re-broadcasting identical bytes is
	// idempotent: the tx hash is unchanged, so existing inclusion/timeout tracking
	// keeps applying and CometBFT dedups duplicates by hash.
	txBz []byte
	// submitHeight is the block height observed when the tx was first broadcast.
	submitHeight int64
	// timeoutHeight is the tx's TimeoutHeight (e.g. proof/claim window close).
	timeoutHeight int64
	// rebroadcasts counts how many times the tx has been re-broadcast so far.
	rebroadcasts int
}

// rebroadcastItem is a unit of re-broadcast work handed out of the locked
// collection step to the (unlocked) broadcast step.
type rebroadcastItem struct {
	txHash string
	txBz   []byte
}

// NewTxClient attempts to construct a new TxClient using the given dependencies
// and options.
//
// It performs the following steps:
//  1. Initializes a default txClient with:
//     - An empty error channel map
//     - An empty transaction timeout pool
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
//   - cometbft/rpc/client.Client
//
// Available options:
//   - WithSigningKeyName
//   - WithConnRetryLimit
//   - WithGasPrices
func NewTxClient(
	ctx context.Context,
	deps depinject.Config,
	opts ...client.TxClientOption,
) (_ client.TxClient, err error) {
	txnClient := &txClient{
		commitTimeoutHeightOffset: DefaultCommitTimeoutHeightOffset,
		txErrorChans:              make(txErrorChansByHash),
		txTimeoutPool:             make(txTimeoutPool),
		rebroadcastPool:           make(map[txHash]*pendingRebroadcast),
	}

	if err = depinject.Inject(
		deps,
		&txnClient.txCtx,
		&txnClient.blockClient,
		&txnClient.logger,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(txnClient)
	}

	// Set the unordered flag on the client context.
	txnClient.txCtx = txnClient.txCtx.WithUnordered(txnClient.unordered)

	if err = txnClient.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	// Form a query based on the client's signing address.
	eventQuery := fmt.Sprintf(txWithSenderAddrQueryFmt, txnClient.signingAddr)

	// Initialize and events replay client.
	txnClient.eventsReplayClient, err = events.NewEventsReplayClient(
		ctx,
		deps,
		eventQuery,
		UnmarshalTxResult,
		defaultTxReplayLimit,
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

// TODO_REFACTOR(red-0ne): Improve transaction error handling architecture
// The current approach returns an async error channel for all transactions, but this
// adds unnecessary complexity. Based on our experience with Cosmos SDK's transaction
// lifecycle, we should separate two distinct error types:
// 1. Mempool inclusion errors - These occur immediately when broadcasting (synchronous)
// 2. Transaction execution errors - These occur during block processing (asynchronous)
// Creating distinct paths for these error types would simplify error handling throughout
// the codebase and improve developer experience.
//
// SignAndBroadcastWithTimeoutHeight signs a set of Cosmos SDK messages, constructs
// a transaction, and broadcasts it to the network. The function performs several
// steps to ensure the messages and the resultant transaction are valid:
//
//  1. Validates each message in the provided set.
//  2. Constructs the transaction using the Cosmos SDK's transaction builder.
//  3. Sets the transaction's timeout height.
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
func (txnClient *txClient) SignAndBroadcastWithTimeoutHeight(
	ctx context.Context,
	timeoutHeight int64,
	msgs ...cosmostypes.Msg,
) (txResponse *cosmostypes.TxResponse, eitherErr either.AsyncError) {
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
		return nil, either.SyncErr(validationErrs)
	}

	// Construct the transactions using cosmos' transactions builder.
	txBuilder := txnClient.txCtx.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		// return synchronous error
		return nil, either.SyncErr(err)
	}

	feeAmount, err := txnClient.getFeeAmount(ctx, txBuilder, msgs...)
	if err != nil {
		return nil, either.SyncErr(err)
	}
	txBuilder.SetFeeAmount(feeAmount)

	txBuilder.SetTimeoutHeight(uint64(timeoutHeight))

	// TODO_TECHDEBT(@bryanchriswhite): Set a timeout timestamp which is estimated
	// to correspond to the timeout height.
	txBuilder.SetTimeoutTimestamp(time.Now().Add(txTimeoutTimestampDelay))

	offline := txnClient.txCtx.GetClientCtx().Offline
	txBuilder.SetUnordered(txnClient.unordered)

	// Override offline mode if unordered is set in order to prevent populating
	// the sequence number. The account number WILL still be queried in TxContext#SignTx().
	if txnClient.unordered {
		offline = true
	}

	// sign transactions
	err = txnClient.txCtx.SignTx(
		txnClient.signingKeyName,
		txBuilder,
		offline, false, txnClient.unordered,
	)
	if err != nil {
		return nil, either.SyncErr(err)
	}

	// ensure transactions is valid
	// NOTE: this makes the transactions valid; i.e. it is *REQUIRED*
	if err = txBuilder.GetTx().ValidateBasic(); err != nil {
		return nil, either.SyncErr(err)
	}

	// serialize transactions
	txBz, err := txnClient.txCtx.EncodeTx(txBuilder)
	if err != nil {
		return nil, either.SyncErr(err)
	}

	// Capture the height at which the tx is broadcast; used to schedule an in-window
	// re-broadcast if the tx is evicted from the mempool before inclusion.
	submitHeight := txnClient.blockClient.LastBlock(ctx).Height()

	txResponse, err = retry.Call(ctx, func() (*cosmostypes.TxResponse, error) {
		response, txErr := txnClient.txCtx.BroadcastTx(txBz)
		// Wrap timeout height error to make it non-retryable.
		if txErr != nil && sdkerrors.ErrTxTimeoutHeight.Is(txErr) {
			txErr = retry.ErrNonRetryable.Wrap(txErr.Error())
		}

		return response, txErr
	}, retry.GetStrategy(ctx), txnClient.logger)
	if err != nil {
		return nil, either.SyncErr(err)
	}

	if txResponse.Code != 0 {
		return txResponse, either.SyncErr(ErrCheckTx.Wrapf("%s", txResponse.RawLog))
	}

	return txResponse, txnClient.addPendingTransactions(
		encoding.NormalizeTxHashHex(txResponse.TxHash),
		txBz,
		submitHeight,
		timeoutHeight,
	)
}

// SignAndBroadcast signs a set of Cosmos SDK messages, constructs a transaction,
// and broadcasts it to the network. The function performs several steps to ensure
// the messages and the resultant transaction are valid:
//
//  1. Validates each message in the provided set.
//  2. Constructs the transaction using the Cosmos SDK's transaction builder.
//  3. Sets the transaction's timeout to the DefaultCommitTimeoutHeightOffset
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
) (txResponse *cosmostypes.TxResponse, eitherErr either.AsyncError) {
	timeoutHeight := txnClient.blockClient.LastBlock(ctx).
		Height() + txnClient.commitTimeoutHeightOffset

	return txnClient.SignAndBroadcastWithTimeoutHeight(
		ctx,
		timeoutHeight,
		msgs...,
	)
}

// validateConfigAndSetDefaults ensures that the necessary configurations for the
// txClient are set, and populates any missing defaults.
//
//  1. It checks if the signing key name is set and returns an error if it's empty.
//  2. It then retrieves the key record from the keyring using the signing key name
//     and checks its existence.
//  3. The address of the signing key is computed and assigned to txClient#signgingAddr.
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

	hasGasSettings := txnClient.gasSetting != nil || txnClient.gasPrices != nil || txnClient.gasAdjustment != 0
	if txnClient.feeAmount != nil && hasGasSettings {
		return fmt.Errorf("cannot set both fee amount and gas settings")
	}

	// Validate gas-related parameters
	if txnClient.feeAmount == nil {
		// If no fee amount is explicitly configured, we need valid gas settings
		if txnClient.gasSetting == nil {
			// Create default gas settings if not provided
			txnClient.gasSetting = &flags.GasSetting{
				Gas:      flags.DefaultGasLimit, // Default gas limit
				Simulate: false,                 // Don't simulate by default
			}
		}

		if txnClient.gasPrices == nil {
			return fmt.Errorf("gas prices must be set when fee amount is not provided")
		}

		// Set the default gas adjustment if simulation is enabled
		if txnClient.gasSetting.Simulate && txnClient.gasAdjustment <= 0 {
			txnClient.gasAdjustment = flags.DefaultGasAdjustment // Common default value
		}
	}

	txnClient.signingAddr = signingAddr

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
	txBz []byte,
	submitHeight int64,
	timeoutHeight int64,
) either.AsyncError {
	txnClient.txsMutex.Lock()
	defer txnClient.txsMutex.Unlock()

	// timeoutHeight is the height that is passed to txBuilder.SetTimeoutHeight
	// - A transaction that is committed at timeoutHeight will be considered valid
	// - txTimeoutPool tracks transactions that are expected to be rejected due to timeout
	txExpirationHeight := timeoutHeight + 1

	// Initialize txTimeoutPool map if necessary.
	txsByHash, ok := txnClient.txTimeoutPool[txExpirationHeight]
	if !ok {
		txsByHash = make(map[string]chan error)
		txnClient.txTimeoutPool[txExpirationHeight] = txsByHash
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

	// Track the signed bytes so the committed-block loop can re-broadcast the tx
	// if it is evicted from the mempool before inclusion. The entry is removed
	// once the tx is observed on-chain (goSubscribeToOwnTxs) or times out
	// (goTimeoutPendingTransactions).
	if _, ok := txnClient.rebroadcastPool[txHash]; !ok {
		txnClient.rebroadcastPool[txHash] = &pendingRebroadcast{
			txBz:          txBz,
			submitHeight:  submitHeight,
			timeoutHeight: timeoutHeight,
		}
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
		txHash := comettypes.Tx(txResult.Tx).Hash()
		txHashHex := encoding.TxHashBytesToNormalizedHex(txHash)

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

			txnClient.logger.Info().Msgf(
				"[TX] Transaction with hash %q committed at height (%d)",
				txHashHex,
				txResult.Height,
			)

			if len(txResult.Result.Log) > 0 {
				txnClient.logger.Error().Msgf(
					"[TX] Transaction with hash %q failed: %s",
					txHashHex,
					txResult.Result.Log,
				)
			}

			// Close and remove from txErrChans
			close(txErrCh)
			delete(txnClient.txErrorChans, txHashHex)
			// Tx is on-chain: stop tracking it for re-broadcast.
			delete(txnClient.rebroadcastPool, txHashHex)
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

		currentHeight := block.Height()

		// Re-broadcast any still-pending txs that are due at this height, to
		// recover from a possible mempool eviction before their timeout height.
		txnClient.rebroadcastDuePendingTxs(currentHeight)

		txnClient.txsMutex.Lock()

		// Retrieve transactions associated with the current block's height.
		txsByHash, ok := txnClient.txTimeoutPool[currentHeight]
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
					panic(fmt.Errorf("expected txErrCh to be closed; received err: %w", err))
				}
				// Remove the processed transaction.
				delete(txsByHash, txHash)
				delete(txnClient.rebroadcastPool, txHash)
				txnClient.txsMutex.Unlock()
				continue
			default:
			}

			// Check if the transaction was not processed by its subscription: handle timeout.
			if err := txnClient.getTxTimeoutError(ctx, currentHeight, txHash); err != nil {
				// Send a tx client timeout error.
				txErrCh <- err
			}
			close(txErrCh)                            // Close the error channel.
			delete(txsByHash, txHash)                 // Remove the transaction.
			delete(txnClient.rebroadcastPool, txHash) // Stop tracking it for re-broadcast.
		}

		// Clean up the txTimeoutPool for the current block height.
		delete(txnClient.txTimeoutPool, currentHeight)
		txnClient.txsMutex.Unlock()
	}
}

// rebroadcastDuePendingTxs re-broadcasts the signed bytes of any still-pending
// transaction whose original broadcast may have been evicted from the mempool
// before inclusion.
//
// A claim/proof tx is broadcast exactly once at its assigned commit height. If it
// is dropped before inclusion (a window-open burst exceeding the block gas limit,
// a mempool recheck eviction, a node restart, ...) it is never re-injected and the
// claim/proof is forfeited at window close (PROOF_MISSING) even when later blocks
// in the window are empty. To recover into those empty tail blocks without
// flooding the mempool with duplicates, each pending tx is re-broadcast at most
// maxTxRebroadcasts times, no earlier than the window midpoint, and not within
// txRebroadcastSafetyBlocks of its timeout height. The signed bytes are identical,
// so the tx hash is unchanged and existing inclusion/timeout tracking still applies.
func (txnClient *txClient) rebroadcastDuePendingTxs(currentHeight int64) {
	for _, item := range txnClient.collectDueRebroadcasts(currentHeight) {
		// Re-broadcasting identical bytes is idempotent: CometBFT dedups by hash,
		// so a "tx already exists in cache" response means the original is still
		// pending and is safe to ignore.
		if _, err := txnClient.txCtx.BroadcastTx(item.txBz); err != nil {
			txnClient.logger.Debug().Err(err).Msgf(
				"[TX] re-broadcast of pending tx %q at height %d returned an error (likely already pending)",
				item.txHash, currentHeight,
			)
			continue
		}
		txnClient.logger.Info().Msgf(
			"[TX] re-broadcast un-included tx %q at height %d to recover from possible mempool eviction",
			item.txHash, currentHeight,
		)
	}
}

// collectDueRebroadcasts returns the pending txs that are due for a re-broadcast
// at currentHeight, incrementing their re-broadcast counter under lock. The actual
// (network-bound) BroadcastTx happens outside the lock in rebroadcastDuePendingTxs.
func (txnClient *txClient) collectDueRebroadcasts(currentHeight int64) []rebroadcastItem {
	txnClient.txsMutex.Lock()
	defer txnClient.txsMutex.Unlock()

	var due []rebroadcastItem
	for hash, pending := range txnClient.rebroadcastPool {
		if pending.rebroadcasts >= maxTxRebroadcasts {
			continue
		}
		// Spread re-broadcasts evenly across the window: the k-th re-broadcast
		// (1-based) is due at submitHeight + window*k/(maxTxRebroadcasts+1). This
		// gives the original tx maximal time to land on its own before the first
		// duplicate, and spaces retries instead of clustering them, so a single
		// congested/empty block does not sink the tx. With maxTxRebroadcasts=1 this
		// reduces to the window midpoint.
		window := pending.timeoutHeight - pending.submitHeight
		slot := window / int64(maxTxRebroadcasts+1)
		nextRebroadcast := int64(pending.rebroadcasts) + 1
		dueHeight := pending.submitHeight + window*nextRebroadcast/int64(maxTxRebroadcasts+1)
		// A whole claim/proof batch is broadcast at window open, so every tx in it
		// shares submitHeight and timeoutHeight and would otherwise re-broadcast on
		// the exact same block, producing a synchronized BroadcastTx/CheckTx burst on
		// the node (observed as a load spike late in the window, including for txs
		// already committed but not yet cleared from the pool by goSubscribeToOwnTxs).
		// Add a deterministic per-tx offset in [0, slot) to fan the batch out across
		// the slot's blocks, then clamp below the safety boundary so the jitter never
		// pushes a tx past the point where a resend can still land.
		dueHeight += rebroadcastJitter(hash, slot)
		if maxDueHeight := pending.timeoutHeight - txRebroadcastSafetyBlocks - 1; dueHeight > maxDueHeight {
			dueHeight = maxDueHeight
		}
		if currentHeight < dueHeight {
			continue
		}
		// Stop once too close to the timeout height for a resend to still land.
		if currentHeight >= pending.timeoutHeight-txRebroadcastSafetyBlocks {
			continue
		}
		pending.rebroadcasts++
		due = append(due, rebroadcastItem{txHash: hash, txBz: pending.txBz})
	}
	return due
}

// rebroadcastJitter returns a stable per-tx block offset in [0, slot) derived from
// the transaction hash. It is used to fan a batch of txs that share the same
// re-broadcast due height (e.g. all claims broadcast at window open) across the
// blocks of a slot instead of firing them on a single block. The offset is a pure
// function of the hash, so a given tx always jitters to the same height across the
// client's lifetime. This is an off-chain scheduling hint only (not consensus
// state), so any stable hash is fine. A slot of 0 or 1 yields no jitter.
func rebroadcastJitter(txHash string, slot int64) int64 {
	if slot <= 1 {
		return 0
	}
	h := fnv.New32a()
	// hash.Hash.Write never returns an error.
	_, _ = h.Write([]byte(txHash))
	return int64(h.Sum32()) % slot
}

// TODO_CONSIDERATION: Simplify error handling by removing custom tx client timeout errors
// We should consider relying solely on CosmosSDK's built-in ErrTxTimeoutHeight error,
// which is already returned by the BroadcastTx() method when a transaction fails to be
// committed before its timeout height. This would eliminate duplicate error handling
// and reduce code complexity.
//
// getTxTimeoutError checks if a transaction with the specified hash has timed out.
// The function decodes the provided hexadecimal hash into bytes and queries the
// transaction using the byte hash. If any error occurs during this process,
// appropriate wrapped errors are returned for easier debugging.
func (txnClient *txClient) getTxTimeoutError(
	ctx context.Context,
	currentHeight int64,
	txHashHex string,
) error {
	// Decode the provided hex hash into bytes.
	txHash, err := hex.DecodeString(txHashHex)
	if err != nil {
		return ErrInvalidTxHash.Wrapf("%s", txHashHex)
	}

	// Query the transaction using the decoded byte hash.
	txResponse, err := txnClient.txCtx.QueryTx(ctx, txHash, false)
	if err != nil || txResponse == nil {
		return ErrQueryTx.Wrapf("with hash: %s: %s", txHashHex, err)
	}

	if txResponse.TxResult.Code != 0 {
		// Return a tx client timeout error with details about the transaction error log.
		return ErrTxTimeout.Wrapf(
			"with tx hash %s and height %d: %s",
			txHashHex, currentHeight, txResponse.TxResult.Log,
		)
	}

	// Transaction was successful even if it was expected to timeout.
	txnClient.logger.Warn().Msgf(
		"expecting tx with hash %s to timeout but was successful at height %d",
		txHashHex, currentHeight,
	)
	return nil

}

// getFeeAmount calculates the transaction fee amount based on client settings.
//
// This method determines the transaction fee using one of two approaches:
// 1. If a fee amount is explicitly set on the client (txnClient.feeAmount), it uses that amount.
// 2. Otherwise, it calculates the fee based on gas limit and gas prices, where:
//   - If simulation is enabled, it estimates gas by simulating the transaction and applies the gas adjustment.
//   - If simulation is disabled, it uses the predefined gas limit from the gas settings.
func (txnClient *txClient) getFeeAmount(
	ctx context.Context,
	txBuilder cosmosclient.TxBuilder,
	msgs ...cosmostypes.Msg,
) (cosmostypes.Coins, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if txnClient.feeAmount != nil {
		// Set the fee amount if provided.
		feeCoins, changeCoins := txnClient.feeAmount.TruncateDecimal()

		// Ensure that any decimal remainder is added to the corresponding coin as an
		// integer amount of the minimal denomination (1upokt).
		// Since changeCoins is the result of DecCoins#TruncateDecimal, it will always
		// be less than 1 unit of the feeCoins.
		if !changeCoins.IsZero() {
			feeCoins = feeCoins.Add(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1))
		}

		return feeCoins, nil
	}

	var gasLimit uint64
	if txnClient.gasSetting.Simulate {
		// If the gas setting is set to simulate, we need to calculate the gas limit
		// based on the messages.
		simulatedGas, err := txnClient.txCtx.GetSimulatedTxGas(ctx, txnClient.signingKeyName, msgs...)
		if err != nil {
			return nil, err
		}
		gasLimit = uint64(float64(simulatedGas) * txnClient.gasAdjustment)
	} else {
		// Otherwise, we use the gas limit from the gas setting.
		gasLimit = txnClient.gasSetting.Gas
	}

	txBuilder.SetGasLimit(gasLimit)

	gasLimitDec := math.LegacyNewDec(int64(gasLimit))
	feeAmountDec := txnClient.gasPrices.MulDec(gasLimitDec)

	feeCoins, changeCoins := feeAmountDec.TruncateDecimal()
	// Ensure that any decimal remainder is added to the corresponding coin as an
	// integer amount of the minimal denomination (1upokt).
	// Since changeCoins is the result of DecCoins#TruncateDecimal, it will always
	// be less than 1 unit of the feeCoins.
	if !changeCoins.IsZero() {
		feeCoins = feeCoins.Add(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1))
	}

	return feeCoins, nil
}

// UnmarshalTxResult extracts an abci.TxResult from a coretypes.ResultEvent.
func UnmarshalTxResult(resultEvt *coretypes.ResultEvent) (*abci.TxResult, error) {
	// Attempt to cast the event data to EventDataTx
	txResult, ok := resultEvt.Data.(comettypes.EventDataTx)
	if !ok {
		return nil, events.ErrEventsUnmarshalEvent.Wrapf("expected EventDataTx, got %T", resultEvt.Data)
	}

	return &txResult.TxResult, nil
}
