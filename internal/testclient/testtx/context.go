package testtx

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	cometbytes "github.com/cometbft/cometbft/libs/bytes"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/internal/testclient"
	"pocket/pkg/client"
	"pocket/pkg/client/tx"
)

// TODO_IMPROVE: these mock constructor helpers could include parameters for the
// "times" (e.g. exact, min, max) values which are passed to their respective
// gomock.EXPECT() method calls (i.e. Times(), MinTimes(), MaxTimes()).
// When implementing such a pattern, be careful about making assumptions about
// correlations between these "times" values and the contexts in which the expected
// methods may be called.

// NewOneTimeErrTxTimeoutTxContext creates a mock transaction context designed to simulate a specific
// timeout error scenario during transaction broadcasting.
//
// Parameters:
// - t: The testing.T instance for the current test.
// - keyring: The Cosmos SDK keyring containing the signer's cryptographic keys.
// - signingKeyName: The name of the key within the keyring to use for signing.
// - expectedTx: A pointer whose value will be set to the expected transaction
// bytes (in hexadecimal format).
// - expectedTxHash: A pointer whose value will be set to the expected
// transaction hash.
// - expectedErrMsg: A pointer whose value will be set to the expected error
// message string.
//
// The function performs the following actions:
// 1. It retrieves the signer's cryptographic key from the provided keyring using the signingKeyName.
// 2. It computes the corresponding address of the signer's key.
// 3. It then formats an error message indicating that the fee payer's address does not exist.
// 4. It creates a base mock transaction context using NewBaseTxContext.
// 5. It sets up the mock behavior for the BroadcastTxSync method to return a specific preset response.
// 6. It also sets up the mock behavior for the QueryTx method to return a specific error response.
func NewOneTimeErrTxTimeoutTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
	expectedErrMsg *string,
) *mockclient.MockTxContext {
	t.Helper()

	signerKey, err := keyring.Key(signingKeyName)
	require.NoError(t, err)

	signerAddr, err := signerKey.GetAddress()
	require.NoError(t, err)

	*expectedErrMsg = fmt.Sprintf(
		"fee payer address: %s does not exist: unknown address",
		signerAddr.String(),
	)

	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		expectedTx,
	)

	// intercept #BroadcastTx() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTx(gomock.Any()).
		DoAndReturn(
			func(txBytes []byte) (*cosmostypes.TxResponse, error) {
				var expectedTxHash cometbytes.HexBytes = comettypes.Tx(txBytes).Hash()
				return &cosmostypes.TxResponse{
					Height: 1,
					TxHash: expectedTxHash.String(),
				}, nil
			},
		).Times(1)

	txCtxMock.EXPECT().QueryTx(
		gomock.AssignableToTypeOf(context.Background()),
		gomock.AssignableToTypeOf([]byte{}),
		gomock.AssignableToTypeOf(false),
	).DoAndReturn(
		func(
			ctx context.Context,
			txHash []byte,
			_ bool,
		) (*cometrpctypes.ResultTx, error) {
			return &cometrpctypes.ResultTx{
				Hash:   txHash,
				Height: 1,
				TxResult: abci.ResponseDeliverTx{
					Code:      1,
					Log:       *expectedErrMsg,
					Codespace: "test_codespace",
				},
				Tx: expectedTx.Bytes(),
			}, nil
		},
	)

	return txCtxMock
}

// NewOneTimeErrCheckTxTxContext creates a mock transaction context to simulate
// a specific error scenario during the ABCI check-tx phase (i.e., during initial
// validation before the transaction is included in the block).
//
// Parameters:
// - t: The testing.T instance for the current test.
// - keyring: The Cosmos SDK keyring containing the signer's cryptographic keys.
// - signingKeyName: The name of the key within the keyring to be used for signing.
// - expectedTx: A pointer whose value will be set to the expected transaction
// bytes (in hexadecimal format).
// - expectedTxHash: A pointer whose value will be set to the expected
// transaction hash.
// - expectedErrMsg: A pointer whose value will be set to the expected error
// message string.
//
// The function operates as follows:
//  1. Retrieves the signer's cryptographic key from the provided keyring based on
//     the signingKeyName.
//  2. Determines the corresponding address of the signer's key.
//  3. Composes an error message suggesting that the fee payer's address is unrecognized.
//  4. Creates a base mock transaction context using the NewBaseTxContext function.
//  5. Sets up the mock behavior for the BroadcastTxSync method to return a specific
//     error response related to the check phase of the transaction.
func NewOneTimeErrCheckTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
	expectedErrMsg *string,
) *mockclient.MockTxContext {
	t.Helper()

	signerKey, err := keyring.Key(signingKeyName)
	require.NoError(t, err)

	signerAddr, err := signerKey.GetAddress()
	require.NoError(t, err)

	*expectedErrMsg = fmt.Sprintf(
		"fee payer address: %s does not exist: unknown address",
		signerAddr.String(),
	)

	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		expectedTx,
	)

	// intercept #BroadcastTx() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTx(gomock.Any()).
		DoAndReturn(
			func(txBytes []byte) (*cosmostypes.TxResponse, error) {
				var expectedTxHash cometbytes.HexBytes = comettypes.Tx(txBytes).Hash()
				return &cosmostypes.TxResponse{
					Height:    1,
					TxHash:    expectedTxHash.String(),
					RawLog:    *expectedErrMsg,
					Code:      1,
					Codespace: "test_codespace",
				}, nil
			},
		).Times(1)

	return txCtxMock
}

// NewOneTimeTxTxContext creates a mock transaction context primed to respond with
// a single successful transaction response. This function facilitates testing by
// ensuring that the BroadcastTxSync method will return a specific, controlled response
// without actually broadcasting the transaction to the network.
//
// Parameters:
// - t: The testing.T instance used for the current test, typically passed from
// the calling test function.
// - keyring: The Cosmos SDK keyring containing the available cryptographic keys.
// - signingKeyName: The name of the key within the keyring used for transaction signing.
// - expectedTx: A pointer whose value will be set to the expected transaction
// bytes (in hexadecimal format).
// - expectedTxHash: A pointer whose value will be set to the expected
// transaction hash.
//
// The function operates as follows:
//  1. Constructs a base mock transaction context using the NewBaseTxContext function.
//  2. Configures the mock behavior for the BroadcastTxSync method to return a pre-defined
//     successful transaction response, ensuring that this behavior will only be triggered once.
func NewOneTimeTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
) *mockclient.MockTxContext {
	t.Helper()

	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		expectedTx,
	)

	// intercept #BroadcastTx() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTx(gomock.Any()).
		DoAndReturn(
			func(txBytes []byte) (*cosmostypes.TxResponse, error) {
				var expectedTxHash cometbytes.HexBytes = comettypes.Tx(txBytes).Hash()
				return &cosmostypes.TxResponse{
					Height: 1,
					TxHash: expectedTxHash.String(),
				}, nil
			},
		).Times(1)

	return txCtxMock
}

// NewBaseTxContext establishes a foundational mock transaction context with
// predefined behaviors suitable for a broad range of testing scenarios. It ensures
// that when interactions like transaction building, signing, and encoding occur
// in the test environment, they produce predictable and controlled outcomes.
//
// Parameters:
// - t: The testing.T instance used for the current test, typically passed from
// the calling test function.
// - signingKeyName: The name of the key within the keyring to be used for
// transaction signing.
// - keyring: The Cosmos SDK keyring containing the available cryptographic keys.
// - expectedTx: A pointer whose value will be set to the expected transaction
// bytes (in hexadecimal format).
// - expectedTxHash: A pointer whose value will be set to the expected
// transaction hash.
// - expectedErrMsg: A pointer whose value will be set to the expected error
// message string.
//
// The function works as follows:
//  1. Invokes the NewAnyTimesTxTxContext to create a base mock transaction context.
//  2. Sets the expectation that NewTxBuilder method will be called exactly once.
//  3. Configures the mock behavior for the SignTx method to utilize the context's
//     signing logic.
//  4. Overrides the EncodeTx method's behavior to intercept the encoding operation,
//     capture the encoded transaction bytes, compute the transaction hash, and populate
//     the expectedTx and expectedTxHash parameters accordingly.
func NewBaseTxContext(
	t *testing.T,
	signingKeyName string,
	keyring cosmoskeyring.Keyring,
	expectedTx *cometbytes.HexBytes,
) *mockclient.MockTxContext {
	t.Helper()

	txCtxMock, txCtx := NewAnyTimesTxTxContext(t, keyring)
	txCtxMock.EXPECT().NewTxBuilder().
		DoAndReturn(txCtx.NewTxBuilder).
		AnyTimes()
	txCtxMock.EXPECT().SignTx(
		gomock.Eq(signingKeyName),
		gomock.AssignableToTypeOf(txCtx.NewTxBuilder()),
		gomock.Eq(false), gomock.Eq(false),
	).DoAndReturn(txCtx.SignTx).AnyTimes()
	txCtxMock.EXPECT().EncodeTx(gomock.Any()).
		DoAndReturn(
			func(txBuilder cosmosclient.TxBuilder) (_ []byte, err error) {
				// intercept cosmosTxContext#EncodeTx to get the encoded tx cometbytes
				*expectedTx, err = txCtx.EncodeTx(txBuilder)
				require.NoError(t, err)
				return expectedTx.Bytes(), nil
			},
		).AnyTimes()

	return txCtxMock
}

// NewAnyTimesTxTxContext initializes a mock transaction context that's configured to allow
// arbitrary calls to certain predefined interactions, primarily concerning the retrieval
// of account numbers and sequences.
//
// Parameters:
// - t: The testing.T instance used for the current test, typically passed from the calling test function.
// - keyring: The Cosmos SDK keyring containing the available cryptographic keys.
//
// The function operates in the following manner:
// 1. Establishes a new gomock controller for setting up mock expectations and behaviors.
// 2. Prepares a set of flags suitable for localnet testing environments.
// 3. Sets up a mock behavior to intercept the GetAccountNumberSequence method calls,
//    ensuring that whenever this method is invoked, it consistently returns an account number
//    and sequence of 1, without making real queries to the underlying infrastructure.
// 4. Constructs a client context tailored for localnet testing with the provided keyring
//    and the mocked account retriever.
// 5. Initializes a transaction factory from the client context and validates its integrity.
// 6. Injects the transaction factory and client context dependencies to create a new transaction context.
// 7. Creates a mock transaction context that always returns the provided keyring when the GetKeyring method is called.
//
// This setup aids tests by facilitating the creation of mock transaction contexts that have predictable
// and controlled outcomes for account number and sequence retrieval operations.
//
// Returns:
// - A mock transaction context suitable for setting additional expectations in tests.
// - A real transaction context initialized with the supplied dependencies.

func NewAnyTimesTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
) (*mockclient.MockTxContext, client.TxContext) {
	t.Helper()

	var (
		ctrl    = gomock.NewController(t)
		flagSet = testclient.NewLocalnetFlagSet(t)
	)

	// intercept #GetAccountNumberSequence() call to mock response and prevent actual query
	accountRetrieverMock := mockclient.NewMockAccountRetriever(ctrl)
	accountRetrieverMock.EXPECT().GetAccountNumberSequence(gomock.Any(), gomock.Any()).
		Return(uint64(1), uint64(1), nil).
		AnyTimes()

	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).
		WithKeyring(keyring).
		WithAccountRetriever(accountRetrieverMock)

	txFactory, err := cosmostx.NewFactoryCLI(clientCtx, flagSet)
	require.NoError(t, err)
	require.NotEmpty(t, txFactory)

	txCtxDeps := depinject.Supply(txFactory, clientCtx)
	txCtx, err := tx.NewTxContext(txCtxDeps)
	require.NoError(t, err)
	txCtxMock := mockclient.NewMockTxContext(ctrl)
	txCtxMock.EXPECT().GetKeyring().Return(keyring).AnyTimes()

	return txCtxMock, txCtx
}
