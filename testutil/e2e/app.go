package e2e

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"

	comettypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/utilities"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/testclient"
)

// E2EApp wraps an integration.App and provides both gRPC and WebSocket servers for end-to-end testing
type E2EApp struct {
	*integration.App
	grpcServer      *grpc.Server
	grpcListener    net.Listener
	wsServer        *http.Server
	wsListener      net.Listener
	wsUpgrader      websocket.Upgrader
	wsConnMutex     sync.RWMutex
	wsConnections   map[*websocket.Conn]map[string]struct{} // maps connections to their subscribed event queries
	resultEventChan chan *coretypes.ResultEvent
}

// NewE2EApp creates a new E2EApp instance with integration.App, gRPC, and WebSocket servers
func NewE2EApp(t *testing.T, opts ...integration.IntegrationAppOptionFn) *E2EApp {
	t.Helper()
	ctx := context.Background()

	// Initialize and start gRPC server
	creds := insecure.NewCredentials()
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	mux := runtime.NewServeMux()

	rootPattern, err := runtime.NewPattern(
		1,
		[]int{int(utilities.OpLitPush), int(utilities.OpNop)},
		[]string{""},
		"",
	)
	require.NoError(t, err)

	// Create the integration app
	opts = append(opts, integration.WithGRPCServer(grpcServer))
	app := integration.NewCompleteIntegrationApp(t, opts...)
	app.RegisterGRPCServer(grpcServer)

	flagSet := testclient.NewFlagSet(t, "tcp://127.0.0.1:42070")
	keyRing := keyring.NewInMemory(app.GetCodec())
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).WithKeyring(keyRing)

	// Register the handler with the mux
	client, err := grpc.NewClient("127.0.0.1:42069", grpc.WithInsecure())
	require.NoError(t, err)

	for _, mod := range app.GetModuleManager().Modules {
		mod.(module.AppModuleBasic).RegisterGRPCGatewayRoutes(clientCtx, mux)
	}

	// Create listeners for gRPC, WebSocket, and HTTP
	grpcListener, err := net.Listen("tcp", "127.0.0.1:42069")
	require.NoError(t, err, "failed to create gRPC listener")

	wsListener, err := net.Listen("tcp", "127.0.0.1:6969")
	require.NoError(t, err, "failed to create WebSocket listener")

	e2eApp := &E2EApp{
		App:             app,
		grpcListener:    grpcListener,
		grpcServer:      grpcServer,
		wsListener:      wsListener,
		wsConnections:   make(map[*websocket.Conn]map[string]struct{}),
		wsUpgrader:      websocket.Upgrader{},
		resultEventChan: make(chan *coretypes.ResultEvent),
	}

	mux.Handle(http.MethodPost, rootPattern, newPostHandler(ctx, client, e2eApp))

	go func() {
		if err := e2eApp.grpcServer.Serve(grpcListener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Initialize and start WebSocket server
	e2eApp.wsServer = newWebSocketServer(e2eApp)
	go func() {
		if err := e2eApp.wsServer.Serve(wsListener); err != nil && errors.Is(err, http.ErrServerClosed) {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Initialize and start HTTP server
	go func() {
		if err := http.ListenAndServe("127.0.0.1:42070", mux); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Start event handling
	go e2eApp.handleResultEvents(t)

	return e2eApp
}

// Close gracefully shuts down the E2EApp and its servers
func (app *E2EApp) Close() error {
	app.grpcServer.GracefulStop()
	if err := app.wsServer.Close(); err != nil {
		return err
	}

	close(app.resultEventChan)

	return nil
}

// GetClientConn returns a gRPC client connection to the E2EApp's gRPC server
func (app *E2EApp) GetClientConn(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx,
		app.grpcListener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// GetWSEndpoint returns the WebSocket endpoint URL
func (app *E2EApp) GetWSEndpoint() string {
	return "ws://" + app.wsListener.Addr().String() + "/websocket"
}

// TODO_IN_THIS_COMMIT: godoc & move...
func (app *E2EApp) GetCometBlockID() comettypes.BlockID {
	lastBlockID := app.GetSdkCtx().BlockHeader().LastBlockId
	partSetHeader := lastBlockID.GetPartSetHeader()

	return comettypes.BlockID{
		Hash: lastBlockID.GetHash(),
		PartSetHeader: comettypes.PartSetHeader{
			Total: partSetHeader.GetTotal(),
			Hash:  partSetHeader.GetHash(),
		},
	}
}
