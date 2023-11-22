package proxy

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayServer = (*synchronousRPCServer)(nil)

// synchronousRPCServer is the struct that holds the state of the synchronous
// RPC server. It is used to listen for and respond to relay requests where
// there is a one-to-one correspondence between relay requests and relay responses.
type synchronousRPCServer struct {
	// service is the service that the server is responsible for.
	service *sharedtypes.Service

	// proxiedServiceEndpoint is the address of the proxied service that the server relays requests to.
	proxiedServiceEndpoint url.URL

	// server is the HTTP server that listens for incoming relay requests.
	server *http.Server

	// relayerProxy is the main relayer proxy that the server uses to perform its operations.
	relayerProxy relayer.RelayerProxy

	// servedRelaysProducer is a channel that emits the relays that have been served, allowing
	// the servedRelays observable to fan-out notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay
}

// NewSynchronousServer creates a new HTTP server that listens for incoming
// relay requests and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments
// and returns a RelayServer that listens to incoming RelayRequests.
func NewSynchronousServer(
	service *sharedtypes.Service,
	supplierEndpointHost string,
	proxiedServiceEndpoint url.URL,
	servedRelaysProducer chan<- *types.Relay,
	proxy relayer.RelayerProxy,
) relayer.RelayServer {
	return &synchronousRPCServer{
		service:                service,
		server:                 &http.Server{Addr: supplierEndpointHost},
		relayerProxy:           proxy,
		proxiedServiceEndpoint: proxiedServiceEndpoint,
		servedRelaysProducer:   servedRelaysProducer,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (sync *synchronousRPCServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		sync.server.Shutdown(ctx)
	}()

	// Set the HTTP handler.
	sync.server.Handler = sync

	return sync.server.ListenAndServe()
}

// Stop terminates the service server and returns an error if it fails.
func (sync *synchronousRPCServer) Stop(ctx context.Context) error {
	return sync.server.Shutdown(ctx)
}

// Service returns the underlying service object.
func (sync *synchronousRPCServer) Service() *sharedtypes.Service {
	return sync.service
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when synchronousRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (sync *synchronousRPCServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	log.Printf("DEBUG: Serving synchronous relay request...")

	// Extract the relay request from the request body.
	log.Printf("DEBUG: Extracting relay request from request body...")
	relayRequest, err := sync.newRelayRequest(request)
	if err != nil {
		sync.replyWithError(relayRequest.Payload, writer, err)
		log.Printf("WARN: failed serving relay request: %s", err)
		return
	}

	// Relay the request to the proxied service and build the response that will be sent back to the client.
	relay, err := sync.serveHTTP(ctx, request, relayRequest)
	if err != nil {
		// Reply with an error if the relay could not be served.
		sync.replyWithError(relayRequest.Payload, writer, err)
		log.Printf("WARN: failed serving relay request: %s", err)
		return
	}

	// Send the relay response to the client.
	if err := sync.sendRelayResponse(relay.Res, writer); err != nil {
		sync.replyWithError(relayRequest.Payload, writer, err)
		log.Printf("WARN: failed sending relay response: %s", err)
		return
	}

	log.Printf(
		"INFO: relay request served successfully for application %s, service %s, session start block height %d, proxied service %s",
		relay.Res.Meta.SessionHeader.ApplicationAddress,
		relay.Res.Meta.SessionHeader.Service.Id,
		relay.Res.Meta.SessionHeader.SessionStartBlockHeight,
		sync.server.Addr,
	)

	// Emit the relay to the servedRelays observable.
	sync.servedRelaysProducer <- relay
}

// serveHTTP holds the underlying logic of ServeHTTP.
func (sync *synchronousRPCServer) serveHTTP(
	ctx context.Context,
	request *http.Request,
	relayRequest *types.RelayRequest,
) (*types.Relay, error) {
	// Verify the relay request signature and session.
	// TODO_TECHDEBT(red-0ne): Currently, the relayer proxy is responsible for verifying
	// the relay request signature. This responsibility should be shifted to the relayer itself.
	// Consider using a middleware pattern to handle non-proxy specific logic, such as
	// request signature verification, session verification, and response signature.
	// This would help in separating concerns and improving code maintainability.
	// See https://github.com/pokt-network/poktroll/issues/160
	if err := sync.relayerProxy.VerifyRelayRequest(ctx, relayRequest, sync.service); err != nil {
		return nil, err
	}

	// Get the relayRequest payload's `io.ReadCloser` to add it to the http.Request
	// that will be sent to the proxied (i.e. staked for) service.
	// (see https://pkg.go.dev/net/http#Request) Body field type.
	requestBodyReader := io.NopCloser(bytes.NewBuffer(relayRequest.Payload))
	log.Printf("DEBUG: Relay request payload: %s", string(relayRequest.Payload))

	// Build the request to be sent to the native service by substituting
	// the destination URL's host with the native service's listen address.
	log.Printf(
		"DEBUG: Building relay request to native service %s...",
		sync.proxiedServiceEndpoint.String(),
	)

	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    &sync.proxiedServiceEndpoint,
		Host:   sync.proxiedServiceEndpoint.Host,
		Body:   requestBodyReader,
	}

	// Send the relay request to the native service.
	httpResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, err
	}

	// Build the relay response from the native service response
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header
	// since it was verified to be valid and has to be the same as the
	// relayResponse session header.
	log.Printf("DEBUG: Building relay response from native service response...")
	relayResponse, err := sync.newRelayResponse(httpResponse, relayRequest.Meta.SessionHeader)
	if err != nil {
		return nil, err
	}

	return &types.Relay{Req: relayRequest, Res: relayResponse}, nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (sync *synchronousRPCServer) sendRelayResponse(
	relayResponse *types.RelayResponse,
	writer http.ResponseWriter,
) error {
	cdc := types.ModuleCdc
	relayResponseBz, err := cdc.Marshal(relayResponse)
	if err != nil {
		return err
	}

	_, err = writer.Write(relayResponseBz)
	return err
}
