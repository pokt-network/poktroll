package testtx

import (
	"testing"

	"cosmossdk.io/depinject"
	cometbytes "github.com/cometbft/cometbft/libs/bytes"
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

func NewOneTimeTxTxContext(
	t *testing.T,
	keyring cosmoskeyring.Keyring,
	signingKeyName string,
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
