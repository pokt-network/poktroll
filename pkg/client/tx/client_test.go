package tx_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	cometbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	"github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

const (
	testSigningKeyName = "test_signer"
	// NB: testServiceIdPrefix must not be longer than 7 characters due to
	// maxServiceIdLen.
	testServiceIdPrefix = "testsvc"
	txCommitTimeout     = 10 * time.Millisecond
)

// TODO_TECHDEBT: add coverage for the transactions client handling an events bytes error either.

func TestTxClient_SignAndBroadcast_Succeeds(t *testing.T) {
	t.Skip("TODO_TECHDEBT(@bryanchriswhite, #425): Revisit Observable test tooling & fix flaky test")
	var (
		// expectedTx is the expected transactions bytes that will be signed and broadcast
		// by the transaction client. It is computed and assigned in the
		// testtx.NewOneTimeTxTxContext helper function. The same reference needs
		// to be used across the expectations that are set on the transactions context mock.
		expectedTx cometbytes.HexBytes
		// txResultsBzPublishChMu is a mutex that protects txResultsBzPublishCh from concurrent access
		// as it is expected to be updated in a mock method but is also sent on in the test.
		txResultsBzPublishChMu = new(sync.Mutex)
		// txResultsBzPublishCh is the channel that the mock events query client
		// will use to publish the transactions event bytes. It is used near the end of
		// the test to mock the network signaling that the transactions was committed.
		txResultsBzPublishCh chan<- either.Bytes
		blocksPublishCh      chan client.Block
		ctx                  = context.Background()
	)

	// Prepare a new test keyring with a test signing key.
	keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	// Construct a new mock events query client
	eventsQueryClient := testeventsquery.NewOneTimeTxEventsQueryClient(
		ctx, t, signingKey, txResultsBzPublishChMu, &txResultsBzPublishCh,
	)

	// Construct a new mock transactions context
	txCtxMock := testtx.NewOneTimeTxTxContext(
		t, keyring,
		testSigningKeyName,
		&expectedTx,
	)

	// Construct a new mock block client (required dependency)
	// - No need to set particular expectations
	// - Contents of the latest block are not relevant
	blockClientMock := testblock.NewOneTimeCommittedBlocksSequenceBlockClient(
		t, blocksPublishCh,
	)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtxMock,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(
		ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName),
	)
	require.NoError(t, err)

	// Get the signing key address.
	signingKeyAddr, err := signingKey.GetAddress()
	require.NoError(t, err)

	// Construct a valid (arbitrary) message to sign, encode, and broadcast.
	// We're using StakeApplication but it could have been any other message type.
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address:  signingKeyAddr.String(),
		Stake:    &appStake,
		Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 1),
	}

	// Sign and broadcast the message.
	eitherErr := txClient.SignAndBroadcast(ctx, appStakeMsg)
	err, errCh := eitherErr.SyncOrAsyncError()
	require.NoError(t, err)

	// Construct the expected RPC response from the expected transaction bytes.
	txResultEvent := &tx.CometTxEvent{}
	txResultEvent.Data.Value.TxResult.Tx = expectedTx
	txResultBz, err := json.Marshal(txResultEvent)
	require.NoError(t, err)
	rpcResult := &rpctypes.RPCResponse{
		Result: txResultBz,
	}
	rpcResultBz, err := json.Marshal(rpcResult)
	require.NoError(t, err)

	// Publish the transaction event bytes to the events query client so that the transaction client
	// registers the transactions as committed (i.e. removes it from the timeout pool).
	txResultsBzPublishChMu.Lock()
	txResultsBzPublishCh <- either.Success[[]byte](rpcResultBz)
	txResultsBzPublishChMu.Unlock()

	// Assert that the error channel was closed without receiving.
	select {
	case err, ok := <-errCh:
		require.NoError(t, err)
		require.Falsef(t, ok, "expected errCh to be closed")
	case <-time.After(txCommitTimeout):
		t.Fatal("test timed out waiting for errCh to receive")
	}
}

func TestTxClient_NewTxClient_Error(t *testing.T) {
	// Construct an empty in-memory keyring.
	memKeyring := cosmoskeyring.NewInMemory(testclient.Marshaler)

	tests := []struct {
		name           string
		signingKeyName string
		expectedErr    error
	}{
		{
			name:           "empty signing key name",
			signingKeyName: "",
			expectedErr:    keyring.ErrEmptySigningKeyName,
		},
		{
			name:           "signing key does not exist",
			signingKeyName: "nonexistent",
			expectedErr:    keyring.ErrNoSuchSigningKey,
		},
		// TODO_TECHDEBT: add coverage for this error case
		// {
		// 	name:        "failed to get address",
		//   testSigningKeyName: "incompatible",
		// 	expectedErr: tx.ErrSigningKeyAddr,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				ctrl = gomock.NewController(t)
				ctx  = context.Background()
			)

			// Construct a new mock events query client. Since we expect the
			// NewTxClient call to fail, we don't need to set any expectations
			// on this mock.
			eventsQueryClient := mockclient.NewMockEventsQueryClient(ctrl)

			// Construct a new mock transactions context.
			txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, memKeyring)

			// Construct a new mock block client. Since we expect the NewTxClient
			// call to fail, we don't need to set any expectations on this mock.
			blockClientMock := mockclient.NewMockBlockClient(ctrl)

			// Construct a new depinject config with the mocks we created above.
			txClientDeps := depinject.Supply(
				eventsQueryClient,
				txCtxMock,
				blockClientMock,
			)

			// Construct a signing key option using the test signing key name.
			signingKeyOpt := tx.WithSigningKeyName(test.signingKeyName)

			// Attempt to create the transactions client.
			txClient, err := tx.NewTxClient(ctx, txClientDeps, signingKeyOpt)
			require.ErrorIs(t, err, test.expectedErr)
			require.Nil(t, txClient)
		})
	}
}

func TestTxClient_SignAndBroadcast_SyncError(t *testing.T) {
	var (
		// txResultsBzPublishChMu is a mutex that protects txResultsBzPublishCh from concurrent access
		// as it is expected to be updated in a mock method but is also sent on in the test.
		txResultsBzPublishChMu = new(sync.Mutex)
		// txResultsBzPublishCh is the channel that the mock events query client
		// will use to publish the transactions event bytes. It is not used in
		// this test but is required to use the NewOneTimeTxEventsQueryClient
		// helper.
		txResultsBzPublishCh chan<- either.Bytes
		// blocksPublishCh is the channel that the mock block client will use
		// to publish the latest block. It is not used in this test but is
		// required to use the NewOneTimeCommittedBlocksSequenceBlockClient
		// helper.
		blocksPublishCh chan client.Block
		ctx             = context.Background()
	)

	keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	// Construct a new mock events query client. Since we expect the
	// NewTxClient call to fail, we don't need to set any expectations
	// on this mock.
	eventsQueryClient := testeventsquery.NewOneTimeTxEventsQueryClient(
		ctx, t, signingKey, txResultsBzPublishChMu, &txResultsBzPublishCh,
	)

	// Construct a new mock transaction context.
	txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, keyring)

	// Construct a new mock block client because it is a required dependency.
	// Since we're not exercising transactions timeouts in this test, we don't need to
	// set any particular expectations on it, nor do we care about the contents
	// of the latest block.
	blockClientMock := testblock.NewOneTimeCommittedBlocksSequenceBlockClient(
		t, blocksPublishCh,
	)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtxMock,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(
		ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName),
	)
	require.NoError(t, err)

	// Construct an invalid (arbitrary) message to sign, encode, and broadcast.
	signingAddr, err := signingKey.GetAddress()
	require.NoError(t, err)
	appStakeMsg := &apptypes.MsgStakeApplication{
		// Providing address to avoid panic from #GetSigners().
		Address: signingAddr.String(),
		Stake:   nil,
		// NB: explicitly omitting required fields
	}

	eitherErr := txClient.SignAndBroadcast(ctx, appStakeMsg)
	err, _ = eitherErr.SyncOrAsyncError()
	require.ErrorIs(t, err, tx.ErrInvalidMsg)

	time.Sleep(10 * time.Millisecond)
}

// TODO_INCOMPLETE: add coverage for async error; i.e. insufficient gas or onchain error
func TestTxClient_SignAndBroadcast_CheckTxError(t *testing.T) {
	var (
		// expectedErrMsg is the expected error message that will be returned
		// by the transaction client. It is computed and assigned in the
		// testtx.NewOneTimeErrCheckTxTxContext helper function.
		expectedErrMsg string
		// txResultsBzPublishChMu is a mutex that protects txResultsBzPublishCh from concurrent access
		// as it is expected to be updated in a mock method but is also sent on in the test.
		txResultsBzPublishChMu = new(sync.Mutex)
		// txResultsBzPublishCh is the channel that the mock events query client
		// will use to publish the transactions event bytes. It is used near the end of
		// the test to mock the network signaling that the transactions was committed.
		txResultsBzPublishCh chan<- either.Bytes
		blocksPublishCh      chan client.Block
		ctx                  = context.Background()
	)

	if os.Getenv("INCLUDE_FLAKY_TESTS") != "true" {
		t.Skip("Skipping known flaky test: 'TestTxClient_SignAndBroadcast_CheckTxError'")
	} else {
		t.Log(`TODO_FLAKY: Running known flaky test: 'TestTxClient_SignAndBroadcast_CheckTxError'

Run the following command a few times to verify it passes at least once:

$ go test -v -count=1 -run TestTxClient_SignAndBroadcast_CheckTxError ./pkg/client/tx/...`)
	}

	keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	eventsQueryClient := testeventsquery.NewOneTimeTxEventsQueryClient(
		ctx, t, signingKey, txResultsBzPublishChMu, &txResultsBzPublishCh,
	)

	txCtxMock := testtx.NewOneTimeErrCheckTxTxContext(
		t, keyring,
		testSigningKeyName,
		&expectedErrMsg,
	)

	// Construct a new mock block client because it is a required dependency.
	// Since we're not exercising transactions timeouts in this test, we don't need to
	// set any particular expectations on it, nor do we care about the contents
	// of the latest block.
	blockClientMock := testblock.NewOneTimeCommittedBlocksSequenceBlockClient(
		t, blocksPublishCh,
	)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtxMock,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName))
	require.NoError(t, err)

	signingKeyAddr, err := signingKey.GetAddress()
	require.NoError(t, err)

	// Construct a valid (arbitrary) message to sign, encode, and broadcast.
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address:  signingKeyAddr.String(),
		Stake:    &appStake,
		Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 1),
	}

	// Sign and broadcast the message.
	eitherErr := txClient.SignAndBroadcast(ctx, appStakeMsg)
	err, _ = eitherErr.SyncOrAsyncError()
	require.ErrorIs(t, err, tx.ErrCheckTx)
	require.ErrorContains(t, err, expectedErrMsg)
}

func TestTxClient_SignAndBroadcast_Timeout(t *testing.T) {
	var (
		timeoutHeight = int64(5)
		// expectedErrMsg is the expected error message that will be returned
		// by the transaction client. It is computed and assigned in the
		// testtx.NewOneTimeErrCheckTxTxContext helper function.
		expectedErrMsg string
		// txResultsBzPublishChMu is a mutex that protects txResultsBzPublishCh from concurrent access
		// as it is expected to be updated in a mock method but is also sent on in the test.
		txResultsBzPublishChMu = new(sync.Mutex)
		// txResultsBzPublishCh is the channel that the mock events query client
		// will use to publish the transaction event bytes. It is used near the end of
		// the test to mock the network signaling that the transaction was committed.
		txResultsBzPublishCh chan<- either.Bytes
		blocksPublishCh      = make(chan client.Block, timeoutHeight)
		ctx                  = context.Background()

		// Trie related variables
		spec           = smt.NewTrieSpec(protocol.NewTrieHasher(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
	)

	keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	eventsQueryClient := testeventsquery.NewOneTimeTxEventsQueryClient(
		ctx, t, signingKey, txResultsBzPublishChMu, &txResultsBzPublishCh,
	)

	txCtxMock := testtx.NewOneTimeErrTxTimeoutTxContext(
		t, keyring,
		testSigningKeyName,
		&expectedErrMsg,
	)

	// Construct a new mock block client because it is a required dependency.
	// Since we're not exercising transaction timeouts in this test, we don't need to
	// set any particular expectations on it, nor do we care about the contents
	// of the latest block.
	blockClientMock := testblock.NewOneTimeCommittedBlocksSequenceBlockClient(
		t, blocksPublishCh,
	)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtxMock,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(
		ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName),
	)
	require.NoError(t, err)

	signingKeyAddr, err := signingKey.GetAddress()
	require.NoError(t, err)

	// Construct a valid (arbitrary) message to sign, encode, and broadcast.
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address:  signingKeyAddr.String(),
		Stake:    &appStake,
		Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 1),
	}

	// Sign and broadcast the message in a transaction.
	eitherErr := txClient.SignAndBroadcast(ctx, appStakeMsg)
	err, errCh := eitherErr.SyncOrAsyncError()
	require.NoError(t, err)

	for i := int64(0); i < timeoutHeight; i++ {
		blocksPublishCh <- testblock.NewAnyTimesBlock(t, emptyBlockHash, int64(i+1))
	}

	// Assert that we receive the expected error type & message.
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, tx.ErrTxTimeout)
		require.ErrorContains(t, err, expectedErrMsg)
	// NB: wait 110% of txCommitTimeout; a bit longer than strictly necessary in
	// order to mitigate flakiness.
	case <-time.After(txCommitTimeout * 110 / 100):
		t.Fatal("test timed out waiting for errCh to receive")
	}

	// Assert that the error channel was closed.
	select {
	case err, ok := <-errCh:
		require.Falsef(t, ok, "expected errCh to be closed")
		require.NoError(t, err)
	// NB: Give the error channel some time to be ready to receive in order to
	// mitigate flakiness.
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected errCh to be closed")
	}
}

func TestTxClient_SignAndBroadcast_Retry(t *testing.T) {
	var (
		// expectedTxBz is the expected transaction bytes that will be signed and broadcast
		// by the transaction client.
		// - Computed and assigned in the testtx.NewOneTimeTxTxContext helper function
		// - Same reference needs to be used across expectations set on the transactions context mock
		expectedTxBz cometbytes.HexBytes

		// txResultsBzPublishCh is the channel for mock events query client to publish transaction event bytes
		// - Not used in this test
		// - Required to use the NewOneTimeTxEventsQueryClient helper
		txResultsBzPublishCh chan<- either.Bytes

		// txResultsBzPublishChMu protects txResultsBzPublishCh from concurrent access
		// - Expected to be updated in a mock method but also sent on in the test
		txResultsBzPublishChMu = new(sync.Mutex)

		// blocksPublishCh is the channel for mock block client to publish the latest block
		// - Not used in this test
		// - Required to use the NewOneTimeCommittedBlocksSequenceBlockClient helper
		blocksPublishCh chan client.Block

		ctx = context.Background()
	)

	keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	// Construct a new mock events query client. Since we expect the
	// NewTxClient call to fail, we don't need to set any expectations
	// on this mock.
	eventsQueryClient := testeventsquery.NewOneTimeTxEventsQueryClient(
		ctx, t, signingKey, txResultsBzPublishChMu, &txResultsBzPublishCh,
	)

	// Instruct the tx client to return an error when submitting a transaction.
	callStatus := &callStatus{
		errorToReturn: fmt.Errorf("test error"),
	}
	txCtxMock := newTxContext(t, ctx,
		testSigningKeyName, keyring, &expectedTxBz, callStatus,
	)

	// Construct a new mock block client (required dependency)
	// - Not exercising transaction timeouts in this test
	// - No need to set particular expectations
	// - Contents of the latest block are not relevant
	blockClientMock := testblock.NewOneTimeCommittedBlocksSequenceBlockClient(
		t, blocksPublishCh,
	)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtxMock,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(
		ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName),
	)
	require.NoError(t, err)

	// Construct an arbitrary message to sign, encode, and broadcast.
	signingAddr, err := signingKey.GetAddress()
	require.NoError(t, err)

	// Construct a valid (arbitrary) message to sign, encode, and broadcast.
	// We're using StakeApplication but it could have been any other message type.
	appStake := types.NewCoin(volatile.DenomuPOKT, math.NewInt(1000000))
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address:  signingAddr.String(), // Providing address to avoid panic from #GetSigners().
		Stake:    &appStake,
		Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 1),
	}

	// Sign and broadcast the message.
	go txClient.SignAndBroadcast(ctx, appStakeMsg)

	// Wait for 5 seconds to allow the retry strategy to perform 4 failing retries.
	time.Sleep(5 * time.Second)

	// All attempts should have failed and zero successful attempts.
	require.Equal(t, 4, callStatus.errorCount)
	require.Equal(t, 0, callStatus.successCount)

	// Instruct the tx client to return a successful response when submitting the transaction
	callStatus.errorToReturn = nil

	// Wait for 5 seconds
	// - Allows the retry strategy to perform a last retry after 4 seconds of waiting time
	time.Sleep(5 * time.Second)

	// The error count should remain the same but the success count should be 1
	require.Equal(t, 4, callStatus.errorCount)
	require.Equal(t, 1, callStatus.successCount)

	// Instruct the tx client to return a non-retryable error when submitting the transaction.
	// This will cause the transaction client to stop retrying and return the error.
	callStatus.errorToReturn = sdkerrors.ErrTxTimeoutHeight.Wrap(fmt.Errorf("test error").Error())

	// Sign and broadcast the message.
	go txClient.SignAndBroadcast(ctx, appStakeMsg)

	// Wait the same amount of time and assert that only one failing attempt was made.
	time.Sleep(5 * time.Second)

	// There should be only one non-retryable error.
	require.Equal(t, 5, callStatus.errorCount)
	require.Equal(t, 1, callStatus.successCount)
}

// TODO_TECHDEBT: add coverage for sending multiple messages simultaneously
func TestTxClient_SignAndBroadcast_MultipleMsgs(t *testing.T) {
	t.SkipNow()
}

func TestTxClient_GasConfig(t *testing.T) {
	var (
		ctx             = context.Background()
		blocksPublishCh = make(chan client.Block, 1)

		// Standard test values
		standardGasPrices = cosmostypes.NewDecCoins(
			cosmostypes.NewDecCoin(volatile.DenomuPOKT, math.NewInt(1000)),
		)
		standardFeeAmount = cosmostypes.NewDecCoins(
			cosmostypes.NewDecCoin(volatile.DenomuPOKT, math.NewInt(10000)),
		)
	)

	tests := []struct {
		name          string
		options       []client.TxClientOption
		expectError   bool
		errorContains string
		validateFee   func(t *testing.T, txBuilder cosmosclient.TxBuilder)
	}{
		{
			name: "no gas params - should fail with error",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
			},
			expectError:   true,
			errorContains: "gas prices must be set",
		},
		{
			name: "only gas prices provided - should use default gas settings",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
				tx.WithGasPrices(&standardGasPrices),
			},
			expectError: false,
			validateFee: func(t *testing.T, txBuilder cosmosclient.TxBuilder) {
				// Default gas * gas price = 200000 * 1000 = 200000000
				feeCoins := txBuilder.GetTx().GetFee()
				require.Equal(t, 1, len(feeCoins))
				require.Equal(t, "200000000", feeCoins[0].Amount.String())
				require.Equal(t, volatile.DenomuPOKT, feeCoins[0].Denom)
			},
		},
		{
			name: "fee amount provided - should override gas settings",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
				tx.WithFeeAmount(&standardFeeAmount),
			},
			expectError: false,
			validateFee: func(t *testing.T, txBuilder cosmosclient.TxBuilder) {
				feeCoins := txBuilder.GetTx().GetFee()
				require.Equal(t, 1, len(feeCoins))
				require.Equal(t, "10000", feeCoins[0].Amount.String())
				require.Equal(t, volatile.DenomuPOKT, feeCoins[0].Denom)
			},
		},
		{
			name: "gas simulation enabled - should calculate gas dynamically",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
				tx.WithGasPrices(&standardGasPrices),
				tx.WithGas(&flags.GasSetting{Gas: 200000, Simulate: true}),
				tx.WithGasAdjustment(1.5),
			},
			expectError: false,
			validateFee: func(t *testing.T, txBuilder cosmosclient.TxBuilder) {
				// Simulated gas (will be 100000 in our mock) * adjustment * price
				// 100000 * 1.5 * 1000 = 150000000
				feeCoins := txBuilder.GetTx().GetFee()
				require.Equal(t, 1, len(feeCoins))
				require.Equal(t, "150000000", feeCoins[0].Amount.String())
				require.Equal(t, volatile.DenomuPOKT, feeCoins[0].Denom)
			},
		},
		{
			name: "decimal remainder in fee - should round up",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
				tx.WithGasPrices(&cosmostypes.DecCoins{
					cosmostypes.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(15001, 4)), // 1.5001 uPOKT
				}),
				tx.WithGas(&flags.GasSetting{Gas: 1000, Simulate: false}),
			},
			expectError: false,
			validateFee: func(t *testing.T, txBuilder cosmosclient.TxBuilder) {
				// 1.5001 * 1000 = 1500.1 with decimal rounding to 1501
				feeCoins := txBuilder.GetTx().GetFee()
				require.Equal(t, 1, len(feeCoins))
				require.Equal(t, "1501", feeCoins[0].Amount.String())
				require.Equal(t, volatile.DenomuPOKT, feeCoins[0].Denom)
			},
		},
		{
			name: "fee amount with decimal - should round up",
			options: []client.TxClientOption{
				tx.WithSigningKeyName(testSigningKeyName),
				tx.WithFeeAmount(&cosmostypes.DecCoins{
					cosmostypes.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1005, 1)), // 100.5 uPOKT
				}),
			},
			expectError: false,
			validateFee: func(t *testing.T, txBuilder cosmosclient.TxBuilder) {
				feeCoins := txBuilder.GetTx().GetFee()
				require.Equal(t, 1, len(feeCoins))
				require.Equal(t, "101", feeCoins[0].Amount.String())
				require.Equal(t, volatile.DenomuPOKT, feeCoins[0].Denom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a new test keyring with a test signing key
			keyring, signingKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)
			ctrl := gomock.NewController(t)

			// Create a mock tx context that will capture the tx builder for inspection
			txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, keyring)
			var txBuilder cosmosclient.TxBuilder

			// Special handling for inspection
			if tt.validateFee != nil {
				txCtxMock.EXPECT().NewTxBuilder().DoAndReturn(func() cosmosclient.TxBuilder {
					txBuilder = testtx.NewMockTxBuilder(ctrl)
					return txBuilder
				}).AnyTimes()

				// For simulation tests
				txCtxMock.EXPECT().GetSimulatedTxGas(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(100000), nil).AnyTimes()

				// Other required methods to pass validation
				txCtxMock.EXPECT().SignTx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				txCtxMock.EXPECT().EncodeTx(gomock.Any()).
					Return([]byte("test-tx"), nil).AnyTimes()
				txCtxMock.EXPECT().BroadcastTx(gomock.Any()).
					Return(&cosmostypes.TxResponse{Code: 0, TxHash: "test-hash"}, nil).AnyTimes()
			}

			// Set up events query client mock
			eventsQueryClient := mockclient.NewMockEventsQueryClient(ctrl)

			// Add expectation for EventsBytes
			eventsQueryClient.EXPECT().EventsBytes(
				gomock.Any(),
				gomock.Any(),
			).DoAndReturn(func(ctx context.Context, query string) (client.EventsBytesObservable, error) {
				obs, _ := channel.NewObservable[either.Bytes]()
				return obs, nil
			}).AnyTimes()

			// Add expectation for Close
			eventsQueryClient.EXPECT().Close().AnyTimes()

			// Set up block client mock with CommittedBlocksSequence expectation
			blockClientMock := mockclient.NewMockBlockClient(ctrl)

			// Setup CommittedBlocksSequence expectation
			blockClientMock.EXPECT().CommittedBlocksSequence(gomock.Any()).DoAndReturn(
				func(ctx context.Context) client.BlockReplayObservable {
					obs, _ := channel.NewReplayObservable[client.Block](ctx, 1,
						channel.WithPublisher(blocksPublishCh))
					return obs
				}).AnyTimes()

			// Create a mock block
			mockBlock := testblock.NewAnyTimesBlock(t, nil, 100)

			// Setup LastBlock expectation
			blockClientMock.EXPECT().LastBlock(gomock.Any()).Return(mockBlock).AnyTimes()

			// Create dependency injection config
			deps := depinject.Supply(
				eventsQueryClient,
				txCtxMock,
				blockClientMock,
			)

			// Create the client
			txClient, err := tx.NewTxClient(ctx, deps, tt.options...)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, txClient)

			// If test includes fee validation
			if tt.validateFee != nil {
				// Create a test message to trigger fee calculation
				signingAddr, _ := signingKey.GetAddress()
				appStake := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000000))
				msg := &apptypes.MsgStakeApplication{
					Address:  signingAddr.String(),
					Stake:    &appStake,
					Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 1),
				}

				// Call SignAndBroadcast to trigger fee calculation
				txClient.SignAndBroadcast(ctx, msg)

				// Validate the fee that was set
				tt.validateFee(t, txBuilder)
			}
		})
	}
}

// newTxContext creates a new mock transactions context for testing.
// It sets up the necessary expectations for signing and broadcasting transactions.
//
// The callStatus parameter is used to instrument and track the number of:
// - Successful calls to the BroadcastTx method
// - Failed calls to the BroadcastTx method
func newTxContext(
	t *testing.T,
	_ context.Context,
	signingKeyName string,
	keyring cosmoskeyring.Keyring,
	expectedTx *cometbytes.HexBytes,
	callStatus *callStatus,
) *mockclient.MockTxContext {
	t.Helper()

	// Construct a new mock transactions context.
	txCtxMock := testtx.NewBaseTxContext(t, signingKeyName, keyring, expectedTx)

	txCtxMock.EXPECT().GetSimulatedTxGas(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(uint64(1), nil).
		AnyTimes()

	txCtxMock.EXPECT().BroadcastTx(gomock.Any()).DoAndReturn(
		func(txBytes []byte) (*cosmostypes.TxResponse, error) {
			if callStatus.errorToReturn != nil {
				callStatus.errorCount++
				return nil, callStatus.errorToReturn
			}
			callStatus.successCount++
			return &cosmostypes.TxResponse{}, nil
		},
	).AnyTimes()

	return txCtxMock
}

// callStatus is a struct that instruments the TxContext mock to track whether
// the BroadcastTx method was called successfully or not.
// It tracks the number of successful and failed calls to the BroadcastTx method.
type callStatus struct {
	successCount  int
	errorCount    int
	errorToReturn error
}
