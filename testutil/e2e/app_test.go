package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/mock/gomock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestGRPCServer2(t *testing.T) {
	grpcServer := grpc.NewServer()
	//gwKeeper := gatewaykeeper.NewKeeper()
	//gwSvc := gatewaykeeper.NewMsgServerImpl(gwKeeper)

	app := integration.NewCompleteIntegrationApp(t)
	app.RegisterGRPCServer(grpcServer)

	mux := runtime.NewServeMux()
	err := http.ListenAndServe(":42070", mux)
	require.NoError(t, err)
	//gatewaytypes.RegisterMsgServer(grpcServer, gwSvc)
	//gatewaytypes.RegisterMsgServer(app.MsgServiceRouter(), gwSvc)

	//gatewaytypes.RegisterQueryHandlerFromEndpoint()

	//reflectionService, err := services.NewReflectionService()
	//require.NoError(t, err)

	//desc, err := reflectionService.FileDescriptors(nil, nil)
	//require.NoError(t, err)

	//app := integration.NewCompleteIntegrationApp(t)
	//grpcServer.RegisterService(desc, app.MsgServiceRouter())
}

func TestSanity(t *testing.T) {
	app := integration.NewCompleteIntegrationApp(t)

	//app.Query(nil, &authtypes.QueryAccountRequest{
	//	Address: "pokt1h04g6njyuv03dhd74a73pyzeadmd8dk7l9tsk8",
	//})

	//app.Query(nil, types2.RequestQuery{
	//	Data:   nil,
	//	Path:   "",
	//	Height: 0,
	//	Prove:  false,
	//})

	ctrl := gomock.NewController(t)
	blockQueryClient := mockclient.NewMockBlockQueryClient(ctrl)
	blockQueryClient.EXPECT().
		Block(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, height *int64) (*cometrpctypes.ResultBlock, error) {
				blockResultMock := &cometrpctypes.ResultBlock{
					Block: &types.Block{
						Header: types.Header{
							Height: 1,
						},
					},
				}
				return blockResultMock, nil
			},
		).AnyTimes()
	deps := depinject.Supply(app.QueryHelper(), blockQueryClient)
	sharedClient, err := query.NewSharedQuerier(deps)
	require.NoError(t, err)

	params, err := sharedClient.GetParams(app.GetSdkCtx())
	require.NoError(t, err)

	t.Logf("shared params: %+v", params)
}

func TestNewE2EApp(t *testing.T) {
	initialHeight := int64(7553)
	// TODO_IN_THIS_COMMIT: does this 👆 need to be reconciled with the internal height of app?

	app := NewE2EApp(t)

	keyRing := keyring.NewInMemory(app.GetCodec())
	rec, err := keyRing.NewAccount(
		"gateway2",
		"suffer wet jelly furnace cousin flip layer render finish frequent pledge feature economy wink like water disease final erase goat include apple state furnace",
		"",
		cosmostypes.FullFundraiserPath,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	gateway2Addr, err := rec.GetAddress()
	require.NoError(t, err)

	// TODO_IN_THIS_COMMOT: fund gateway2 account.
	_, err = app.RunMsg(t, &banktypes.MsgSend{
		FromAddress: app.GetFaucetBech32(),
		ToAddress:   gateway2Addr.String(),
		Amount:      cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100000000)),
	})
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	blockQueryClient := mockclient.NewMockBlockQueryClient(ctrl)
	blockQueryClient.EXPECT().
		Block(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, height *int64) (*cometrpctypes.ResultBlock, error) {
				//time.Sleep(time.Second * 100)
				blockResultMock := &cometrpctypes.ResultBlock{
					Block: &types.Block{
						Header: types.Header{
							Height: initialHeight,
						},
					},
				}
				return blockResultMock, nil
			},
		).AnyTimes()
	//blockQueryClient, err := sdkclient.NewClientFromNode("tcp://127.0.0.1:42070")
	//blockQueryClient, err := sdkclient.NewClientFromNode("tcp://127.0.0.1:26657")
	//require.NoError(t, err)

	deps := depinject.Supply(app.QueryHelper(), blockQueryClient)

	sharedQueryClient, err := query.NewSharedQuerier(deps)
	require.NoError(t, err)

	sharedParams, err := sharedQueryClient.GetParams(app.GetSdkCtx())
	require.NoError(t, err)

	t.Logf("shared params: %+v", sharedParams)

	eventsQueryClient := events.NewEventsQueryClient("ws://127.0.0.1:6969/websocket")
	//eventsQueryClient := events.NewEventsQueryClient("ws://127.0.0.1:26657/websocket")
	deps = depinject.Configs(deps, depinject.Supply(eventsQueryClient))
	blockClient, err := block.NewBlockClient(app.GetSdkCtx(), deps)
	require.NoError(t, err)

	// TODO_IN_THIS_COMMIT: NOT localnet flagset NOR context, should be
	// configured to match the E2E app listeners.
	flagSet := testclient.NewFlagSet(t, "127.0.0.1:42069")
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).WithKeyring(keyRing)

	txFactory, err := cosmostx.NewFactoryCLI(clientCtx, flagSet)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(txtypes.Context(clientCtx), txFactory))

	//_, txContext := testtx.NewE2ETxContext(t, keyRing, flagSet)
	txContext, err := tx.NewTxContext(deps)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(blockClient, txContext))
	txClient, err := tx.NewTxClient(app.GetSdkCtx(), deps, tx.WithSigningKeyName("gateway2"))
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	eitherErr := txClient.SignAndBroadcast(
		app.GetSdkCtx(),
		gatewaytypes.NewMsgStakeGateway(
			"pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
			cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(100000001)),
		),
	)

	// TODO_IN_THIS_COMMIT: signal to the WS server to send another block result event...
	//app.NextBlock(t)

	err, errCh := eitherErr.SyncOrAsyncError()
	require.NoError(t, err)
	require.NoError(t, <-errCh)
}

func TestGRPCServer(t *testing.T) {
	app := NewE2EApp(t)
	t.Cleanup(func() {
		app.Close()
	})

	creds := insecure.NewCredentials()
	grpcConn, err := grpc.NewClient("127.0.0.1:42069", grpc.WithTransportCredentials(creds))
	require.NoError(t, err)

	//dataHex, err := hex.DecodeString("0A2B706F6B74313577336668667963306C747476377235383565326E6370663674326B6C3975683872736E797A")
	require.NoError(t, err)

	req := gatewaytypes.QueryGetGatewayRequest{
		Address: "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
	}

	// convert the request to a proto message
	anyReq, err := codectypes.NewAnyWithValue(&req)
	require.NoError(t, err)

	res := new(gatewaytypes.QueryGetGatewayResponse)

	err = grpcConn.Invoke(context.Background(), "/poktroll.gateway.Query/Gateway", anyReq, res)
	require.NoError(t, err)

	//"method" : "abci_query",
	//"params" : {
	//  "data" : "0A2B706F6B74313577336668667963306C747476377235383565326E6370663674326B6C3975683872736E797A",
	//	"height" : "0",
	//	"path" : "/cosmos.auth.v1beta1.Query/Account",
	//	"prove" : false
	//}

	//"method" : "broadcast_tx_async",
	//"params" : {
	//	"tx" : "CmsKZgohL3Bva3Ryb2xsLmdhdGV3YXkuTXNnU3Rha2VHYXRld2F5EkEKK3Bva3QxNXczZmhmeWMwbHR0djdyNTg1ZTJuY3BmNnQya2w5dWg4cnNueXoSEgoFdXBva3QSCTEwMDAwMDAwMRiGOxJYCk4KRgofL2Nvc21vcy5jcnlwdG8uc2VjcDI1NmsxLlB1YktleRIjCiEDZo2bY9XquUsFljtW/OKWVCDhYFf7NbidN4Y99VQ9438SBAoCCAESBhCqoYLJAhpAw5e7iJN5SpFit3fftxnZY7EDiFqupi7XEL3sUyeV0IBSQv2JZ7Cdu0dCG0yEVgj0xarkPi7dR10pNDL1gcUJxw=="
	//}
}
