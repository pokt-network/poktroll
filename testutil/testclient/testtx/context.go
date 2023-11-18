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

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient"
)

// NewLocalnetContext creates and returns a new transaction context configured
// for use with the localnet sequencer.
func NewLocalnetContext(t *testing.T) client.TxContext {
	t.Helper()

	flagSet := testclient.NewLocalnetFlagSet(t)
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet)
	txFactory, err := cosmostx.NewFactoryCLI(*clientCtx, flagSet)
	require.NoError(t, err)
	require.NotEmpty(t, txFactory)

	deps := depinject.Supply(
		*clientCtx,
		txFactory,
	)

	txCtx, err := tx.NewTxContext(deps)
	require.NoError(t, err)

	return txCtx
}

// TODO_IMPROVE: these mock constructor helpers could include parameters for the
// "times" (e.g. exact, min, max) values which are passed to their respective
// gomock.EXPECT() method calls (i.e. Times(), MinTimes(), MaxTimes()).
// When implementing such a pattern, be careful about making assumptions about
// correlations between these "times" values and the contexts in which the expected
// methods may be called.

// NewOneTimeErrTxTimeoutTxContext creates a mock transaction context designed to
// simulate a specific timeout error scenario during transaction broadcasting.
// expectedErrMsg is populated with the same error message which is presented in
// the result from the QueryTx method so that it can be asserted against.
func NewOneTimeErrTxTimeoutTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
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

	var expectedTx cometbytes.HexBytes
	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		&expectedTx,
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
// expectedErrMsg is populated with the same error message which is presented in
// the result from the QueryTx method so that it can be asserted against.
func NewOneTimeErrCheckTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
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

	var expectedTx cometbytes.HexBytes
	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		&expectedTx,
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
// a single successful transaction response.
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

// NewBaseTxContext creates a mock transaction context that's configured to expect
// calls to NewTxBuilder, SignTx, and EncodeTx methods, any number of times.
// EncodeTx is used to intercept the encoded transaction bytes and store them in
// the expectedTx output parameter. Each of these methods proxies to the corresponding
// method on a real transaction context.
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

	txClientCtx := relayer.TxClientContext(clientCtx)
	txCtxDeps := depinject.Supply(txFactory, txClientCtx)
	txCtx, err := tx.NewTxContext(txCtxDeps)
	require.NoError(t, err)
	txCtxMock := mockclient.NewMockTxContext(ctrl)
	txCtxMock.EXPECT().GetKeyring().Return(keyring).AnyTimes()

	return txCtxMock, txCtx
}
