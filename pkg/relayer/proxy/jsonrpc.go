package proxy

import (
	"context"
	"net/http"
	"net/url"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ RelayServer = (*jsonRPCServer)(nil)

type jsonRPCServer struct {
	// service is the id of the service that the server is responsible for.
	service *sharedtypes.Service

	// serverEndpoint is the advertised endpoint configuration that the server uses to
	// listen for incoming relay requests.
	serverEndpoint *sharedtypes.SupplierEndpoint

	// proxiedServiceEndpoint is the address of the proxied service that the server relays requests to.
	proxiedServiceEndpoint url.URL

	// server is the http server that listens for incoming relay requests.
	server *http.Server

	// relayerProxy is the main relayer proxy that the server uses to perform its operations.
	relayerProxy RelayerProxy

	// servedRelaysProducer is a channel that emits the relays that have been served so that the
	// servedRelays observable can fan out the notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay
}

// NewJSONRPCServer creates a new HTTP server that listens for incoming relay requests
// and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments and returns
// a RelayServer that listens to incoming RelayRequests
func NewJSONRPCServer(
	service *sharedtypes.Service,
	supplierEndpoint *sharedtypes.SupplierEndpoint,
	proxiedServiceEndpoint url.URL,
	servedRelaysProducer chan<- *types.Relay,
	proxy RelayerProxy,
) RelayServer {
	return &jsonRPCServer{
		service:                service,
		serverEndpoint:         supplierEndpoint,
		server:                 &http.Server{Addr: supplierEndpoint.Url},
		relayerProxy:           proxy,
		proxiedServiceEndpoint: proxiedServiceEndpoint,
		servedRelaysProducer:   servedRelaysProducer,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (j *jsonRPCServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		j.server.Shutdown(ctx)
	}()

	return j.server.ListenAndServe()
}

// Stop terminates the service server and returns an error if it fails.
func (j *jsonRPCServer) Stop(ctx context.Context) error {
	return j.server.Shutdown(ctx)
}

// Service returns the JSON-RPC service.
func (j *jsonRPCServer) Service() *sharedtypes.Service {
	return j.service
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when jsonRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (j *jsonRPCServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	panic("TODO: implement jsonRPCServer.ServeHTTP")
}
