package faucet_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/faucet"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
)

const (
	testListenAddress   = "127.0.0.1:42069"
	testTimeoutDuration = time.Second
	mockTxHash          = "0000000000000000000000000000000000000000000000000000000000000000"
	testSendUPOKT       = "100000000000upokt"
	testSendMACT        = "1mact"
	testFeeUPOKT        = "1upokt"

	testSigningKeyName     = "faucet"
	testSigningKeyMnemonic = "baby advance work soap slow exclude blur humble lucky rough teach wide chuckle captain rack laundry butter main very cannon donate armor dress follow"
)

var (
	clientCtx          cosmosclient.Context
	testSigningAddress cosmostypes.AccAddress
)

func TestMain(m *testing.M) {
	cmd.InitSDKConfig()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	// ABSOLUTELY REQUIRED; otherwise, keyring can't (un)marshal...
	cryptocodec.RegisterInterfaces(registry)
	keyRing := keyring.NewInMemory(cdc)

	keyRecord, err := keyRing.NewAccount(
		testSigningKeyName,
		testSigningKeyMnemonic,
		cosmostypes.FullFundraiserPath,
		keyring.DefaultBIP39Passphrase,
		hd.Secp256k1,
	)
	if err != nil {
		panic(err)
	}

	testSigningAddress, err = keyRecord.GetAddress()
	if err != nil {
		panic(err)
	}

	clientCtx = cosmosclient.Context{}.WithKeyring(keyRing)

	m.Run()
}

func TestNewFaucet(t *testing.T) {
	// Ensure the CLI logger is set up.
	logger.LogOutput = flags.DefaultLogOutput
	err := logger.PreRunESetup(nil, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())

	config, err := faucet.NewFaucetConfig(
		clientCtx,
		testSigningKeyName,
		testListenAddress,
		[]string{testSendMACT, testSendUPOKT},
		false,
	)
	require.NoError(t, err)

	msgsPerTx := new([][]cosmostypes.Msg)
	*msgsPerTx = make([][]cosmostypes.Msg, 0)
	signAndBroadcastSuccess := newSignAndBroadcastSuccess(t, msgsPerTx)
	txClient := newTxClientMock(t, signAndBroadcastSuccess, 2)

	testRecipientAddress := cosmostypes.MustAccAddressFromBech32(sample.AccAddressBech32())
	ctrl := gomock.NewController(t)
	bankQueryClient := mockclient.NewMockBankGRPCQueryClient(ctrl)

	faucet, err := faucet.NewFaucetServer(
		ctx,
		faucet.WithConfig(config),
		faucet.WithTxClient(txClient),
		faucet.WithBankQueryClient(bankQueryClient),
	)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		asyncErr := faucet.Serve(ctx)
		errCh <- asyncErr
	}()

	// Wait a tick for the faucet to start listening.
	time.Sleep(100 * time.Millisecond)

	t.Run("supported coin #1 (100000000000upokt)", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s/upokt/%s", config.ListenAddress, testRecipientAddress)
		res, getErr := http.DefaultClient.Post(requestURL, "application/json", nil)
		require.NoError(t, getErr)
		require.Equal(t, http.StatusAccepted, res.StatusCode)

		expectedSendCoinsUPOKT, _ := cosmostypes.ParseCoinsNormalized(testSendUPOKT)
		expectedSendMsg := bank.NewMsgSend(
			config.GetSigningAddress(),
			testRecipientAddress,
			expectedSendCoinsUPOKT,
		)
		require.Equal(t, 1, len(*msgsPerTx))
		require.Equal(t, 1, len((*msgsPerTx)[0]))
		require.Equal(t, expectedSendMsg, (*msgsPerTx)[0][0])
	})

	// Reset the msgsPerTx slice to empty
	*msgsPerTx = make([][]cosmostypes.Msg, 0)

	t.Run("supported coin #2 (1mact)", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s/mact/%s", config.ListenAddress, testRecipientAddress)
		res, getErr := http.DefaultClient.Post(requestURL, "application/json", nil)
		require.NoError(t, getErr)
		require.Equal(t, http.StatusAccepted, res.StatusCode)

		expectedSendCoinsMACT, _ := cosmostypes.ParseCoinsNormalized(testSendMACT)
		expectedSendMsg := bank.NewMsgSend(
			config.GetSigningAddress(),
			testRecipientAddress,
			expectedSendCoinsMACT,
		)
		require.Equal(t, 1, len(*msgsPerTx))
		require.Equal(t, 1, len((*msgsPerTx)[0]))
		require.Equal(t, expectedSendMsg, (*msgsPerTx)[0][0])
	})

	cancel()

	select {
	case <-time.After(testTimeoutDuration):
		t.Fatal("Timed out waiting for faucet to shutdown")

	case err = <-errCh:
		require.NoError(t, err)
	}
}

// signAndBroadcastFn is a function which signs and broadcasts the given msgs.
type signAndBroadcastFn func(context.Context, ...cosmostypes.Msg) (*cosmostypes.TxResponse, either.AsyncError)

// newSignAndBroadcastSuccess returns a signAndBroadcastFn which will append
// the given msgs to the given sendTxs slice and return a successful
// cosmostypes.TxResponse.
func newSignAndBroadcastSuccess(t *testing.T, sendTxs *[][]cosmostypes.Msg) signAndBroadcastFn {
	t.Helper()

	return func(
		ctx context.Context,
		msgs ...cosmostypes.Msg,
	) (*cosmostypes.TxResponse, either.AsyncError) {
		*sendTxs = append(*sendTxs, msgs)
		txResponse := &cosmostypes.TxResponse{
			Code:   0,
			TxHash: mockTxHash,
			RawLog: "",
		}
		errCh := make(chan error)
		close(errCh)
		return txResponse, either.AsyncErr(errCh)
	}
}

// newTxClientMock returns a mock client.TxClient which implements the given
// signAndBroadcastFn and expects it to be called times times.
func newTxClientMock(t *testing.T, signAndBroadcast signAndBroadcastFn, times int) client.TxClient {
	t.Helper()

	ctrl := gomock.NewController(t)
	txClient := mockclient.NewMockTxClient(ctrl)
	txClient.EXPECT().SignAndBroadcast(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(signAndBroadcast).Times(times)

	return txClient
}
