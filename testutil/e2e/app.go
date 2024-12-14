package e2e

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"

	"github.com/pokt-network/poktroll/testutil/integration"
)

// E2EApp wraps an integration.App and provides both gRPC and WebSocket servers for end-to-end testing
type E2EApp struct {
	*integration.App
	grpcServer     *grpc.Server
	grpcListener   net.Listener
	wsServer       *http.Server
	wsListener     net.Listener
	httpServer     *http.Server
	httpListener   net.Listener
	wsUpgrader     websocket.Upgrader
	wsConnections  map[*websocket.Conn]map[string]struct{} // maps connections to their subscribed event queries
	wsConnMutex    sync.RWMutex
	blockEventChan chan *coretypes.ResultEvent
}

// NewE2EApp creates a new E2EApp instance with integration.App, gRPC, and WebSocket servers
func NewE2EApp(t *testing.T, opts ...integration.IntegrationAppOptionFn) *E2EApp {
	t.Helper()

	// Create the integration app
	app := integration.NewCompleteIntegrationApp(t, opts...)

	// Create listeners for gRPC, WebSocket, and HTTP
	grpcListener, err := net.Listen("tcp", "localhost:42069")
	require.NoError(t, err, "failed to create gRPC listener")

	wsListener, err := net.Listen("tcp", "localhost:6969")
	require.NoError(t, err, "failed to create WebSocket listener")

	httpListener, err := net.Listen("tcp", "localhost:42070")
	require.NoError(t, err, "failed to create HTTP listener")

	e2eApp := &E2EApp{
		App:            app,
		grpcListener:   grpcListener,
		wsListener:     wsListener,
		httpListener:   httpListener,
		wsConnections:  make(map[*websocket.Conn]map[string]struct{}),
		wsUpgrader:     websocket.Upgrader{},
		blockEventChan: make(chan *coretypes.ResultEvent, 1),
	}

	// Initialize and start gRPC server
	e2eApp.grpcServer = newGRPCServer(e2eApp, t)
	go func() {
		if err := e2eApp.grpcServer.Serve(grpcListener); err != nil {
			panic(err)
		}
	}()

	// Initialize and start WebSocket server
	e2eApp.wsServer = newWebSocketServer(e2eApp)
	go func() {
		if err := e2eApp.wsServer.Serve(wsListener); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Initialize and start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", e2eApp.handleHTTP)
	e2eApp.httpServer = &http.Server{Handler: mux}
	go func() {
		if err := e2eApp.httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
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
	if err := app.httpServer.Close(); err != nil {
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

// handleHTTP handles incoming HTTP requests by responding with RPCResponse
func (app *E2EApp) handleHTTP(w http.ResponseWriter, r *http.Request) {
	var req rpctypes.RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the request - for now just return a basic response
	// TODO_IMPROVE: Implement proper CometBFT RPC endpoint handling
	response := rpctypes.RPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  json.RawMessage(`{}`),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
