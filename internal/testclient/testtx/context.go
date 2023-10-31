package testtx

import (
	"context"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"testing"

	"cosmossdk.io/depinject"
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

func NewOneTimeErrTxTimeoutTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
	expectedTxHash *cometbytes.HexBytes,
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
		expectedTxHash,
	)

	// intercept #BroadcastTxSync() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTxSync(gomock.Any()).
		DoAndReturn(func(txBytes []byte) (*cosmostypes.TxResponse, error) {
			return &cosmostypes.TxResponse{
				Height:    1,
				TxHash:    expectedTxHash.String(),
				RawLog:    "",
				Logs:      nil,
				Tx:        nil,
				Timestamp: "",
				Events:    nil,
			}, nil
		}).Times(1)

	txCtxMock.EXPECT().QueryTx(
		gomock.AssignableToTypeOf(context.Background()),
		gomock.AssignableToTypeOf([]byte{}),
		gomock.AssignableToTypeOf(false),
	).DoAndReturn(func(
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
	})

	return txCtxMock
}

func NewOneTimeErrCheckTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
	expectedTxHash *cometbytes.HexBytes,
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
		expectedTxHash,
	)

	// intercept #BroadcastTxSync() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTxSync(gomock.Any()).
		DoAndReturn(func(txBytes []byte) (*cosmostypes.TxResponse, error) {
			return &cosmostypes.TxResponse{
				Height:    1,
				TxHash:    expectedTxHash.String(),
				RawLog:    *expectedErrMsg,
				Code:      1,
				Codespace: "test_codespace",
				Logs:      nil,
				Tx:        nil,
				Timestamp: "",
				Events:    nil,
			}, nil
		}).Times(1)

	return txCtxMock
}

func NewOneTimeTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
	expectedTx *cometbytes.HexBytes,
	expectedTxHash *cometbytes.HexBytes,
) *mockclient.MockTxContext {
	t.Helper()

	txCtxMock := NewBaseTxContext(
		t, signingKeyName,
		keyring,
		expectedTx,
		expectedTxHash,
	)

	// intercept #BroadcastTxSync() call to mock response and prevent actual broadcast
	txCtxMock.EXPECT().BroadcastTxSync(gomock.Any()).
		DoAndReturn(func(txBytes []byte) (*cosmostypes.TxResponse, error) {
			return &cosmostypes.TxResponse{
				Height:    1,
				TxHash:    expectedTxHash.String(),
				RawLog:    "",
				Logs:      nil,
				Tx:        nil,
				Timestamp: "",
				Events:    nil,
			}, nil
		}).Times(1)

	return txCtxMock
}

func NewBaseTxContext(
	t *testing.T,
	signingKeyName string,
	keyring cosmoskeyring.Keyring,
	expectedTx *cometbytes.HexBytes,
	expectedTxHash *cometbytes.HexBytes,
) *mockclient.MockTxContext {
	t.Helper()

	txCtxMock, txCtx := NewAnyTimesTxTxContext(t, keyring)
	txCtxMock.EXPECT().NewTxBuilder().
		DoAndReturn(txCtx.NewTxBuilder).
		Times(1)
	txCtxMock.EXPECT().SignTx(
		gomock.Eq(signingKeyName),
		gomock.AssignableToTypeOf(txCtx.NewTxBuilder()),
		gomock.Eq(false), gomock.Eq(false),
	).DoAndReturn(txCtx.SignTx).Times(1)
	txCtxMock.EXPECT().EncodeTx(gomock.Any()).
		DoAndReturn(func(txBuilder cosmosclient.TxBuilder) (_ []byte, err error) {
			// intercept cosmosTxContext#EncodeTx to get the encoded tx cometbytes
			*expectedTx, err = txCtx.EncodeTx(txBuilder)
			*expectedTxHash = comettypes.Tx(expectedTx.Bytes()).Hash()
			require.NoError(t, err)
			return expectedTx.Bytes(), nil
		}).Times(1)

	return txCtxMock
}

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
