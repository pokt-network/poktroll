package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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
	grpcServer     *grpc.Server
	grpcListener   net.Listener
	wsServer       *http.Server
	wsListener     net.Listener
	wsUpgrader     websocket.Upgrader
	wsConnMutex    sync.RWMutex
	wsConnections  map[*websocket.Conn]map[string]struct{} // maps connections to their subscribed event queries
	blockEventChan chan *coretypes.ResultEvent
}

// NewE2EApp creates a new E2EApp instance with integration.App, gRPC, and WebSocket servers
func NewE2EApp(t *testing.T, opts ...integration.IntegrationAppOptionFn) *E2EApp {
	t.Helper()

	// Initialize and start gRPC server
	creds := insecure.NewCredentials()
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	mux := runtime.NewServeMux()

	// Create the integration app
	app := integration.NewCompleteIntegrationApp(t, opts...)
	app.RegisterGRPCServer(grpcServer)
	//app.RegisterGRPCServer(e2eApp.grpcServer)

	flagSet := testclient.NewFlagSet(t, "tcp://127.0.0.1:42069")
	keyRing := keyring.NewInMemory(app.GetCodec())
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).WithKeyring(keyRing)

	for moduleName, mod := range app.GetModuleManager().Modules {
		fmt.Printf(">>> %s\n", moduleName)
		mod.(module.AppModuleBasic).RegisterGRPCGatewayRoutes(clientCtx, mux)
	}

	// Create listeners for gRPC, WebSocket, and HTTP
	grpcListener, err := net.Listen("tcp", "localhost:42069")
	require.NoError(t, err, "failed to create gRPC listener")

	wsListener, err := net.Listen("tcp", "localhost:6969")
	require.NoError(t, err, "failed to create WebSocket listener")

	e2eApp := &E2EApp{
		App:            app,
		grpcListener:   grpcListener,
		grpcServer:     grpcServer,
		wsListener:     wsListener,
		wsConnections:  make(map[*websocket.Conn]map[string]struct{}),
		wsUpgrader:     websocket.Upgrader{},
		blockEventChan: make(chan *coretypes.ResultEvent, 1),
	}

	go func() {
		if err := e2eApp.grpcServer.Serve(grpcListener); err != nil {
			panic(err)
		}
	}()

	// Initialize and start WebSocket server
	e2eApp.wsServer = newWebSocketServer(e2eApp)
	go func() {
		if err := e2eApp.wsServer.Serve(wsListener); err != nil && errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Initialize and start HTTP server
	go func() {
		if err := http.ListenAndServe("localhost:42070", mux); err != nil {
			panic(err)
		}
	}()

	// Start event handling
	go e2eApp.handleBlockEvents(t)

	return e2eApp
}

// Close gracefully shuts down the E2EApp and its servers
func (app *E2EApp) Close() error {
	app.grpcServer.GracefulStop()
	if err := app.wsServer.Close(); err != nil {
		return err
	}

	close(app.blockEventChan)

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
