package proxy

import (
	"context"
	"net/http"
)

var _ RelayServer = &jsonRPCServer{}

type jsonRPCServer struct {
	// serviceId is the id of the service that the server is responsible for.
	serviceId string

	// endpointUrl is the url that the server listens to for incoming relay requests.
	endpointUrl string

	// server is the http server that listens for incoming relay requests.
	server *http.Server

	// relayerProxy is the main relayer proxy that the server uses to perform its operations.
	relayerProxy RelayerProxy
}

// NewJSONRPCServer creates a new HTTP server that listens for incoming relay requests
// and proxies them to the supported native service.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments and returns
// a RelayServer that listens to incoming RelayRequests
func NewJSONRPCServer(serviceId string, endpointUrl string, proxy RelayerProxy) RelayServer {
	return &jsonRPCServer{
		serviceId:    serviceId,
		endpointUrl:  endpointUrl,
		server:       &http.Server{Addr: endpointUrl},
		relayerProxy: proxy,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to be done to shut down.
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

// ServiceId returns the serviceId of the service.
func (j *jsonRPCServer) ServiceId() string {
	return j.serviceId
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when jsonRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (j *jsonRPCServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
}
