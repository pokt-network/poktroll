package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"pocket/x/service/types"
	sharedtypes "pocket/x/shared/types"
)

var _ RelayServer = (*jsonRPCServer)(nil)

type jsonRPCServer struct {
	// serviceId is the id of the service that the server is responsible for.
	serviceId *sharedtypes.ServiceId

	// serverEndpoint is the advertised endpoint configuration that the server uses to
	// listen for incoming relay requests.
	serverEndpoint *sharedtypes.SupplierEndpoint

	// proxiedServiceEndpoint is the address of the proxied service that the server relays requests to.
	proxiedServiceEndpoint url.URL

	// server is the HTTP server that listens for incoming relay requests.
	server *http.Server

	// relayerProxy is the main relayer proxy that the server uses to perform its operations.
	relayerProxy RelayerProxy

	// servedRelaysProducer is a channel that emits the relays that have been served, allowing
	// the servedRelays observable to fan-out notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay
}

// NewJSONRPCServer creates a new HTTP server that listens for incoming relay requests
// and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments and returns
// a RelayServer that listens to incoming RelayRequests.
func NewJSONRPCServer(
	serviceId *sharedtypes.ServiceId,
	supplierEndpoint *sharedtypes.SupplierEndpoint,
	proxiedServiceEndpoint url.URL,
	servedRelaysProducer chan<- *types.Relay,
	proxy RelayerProxy,
) RelayServer {
	return &jsonRPCServer{
		serviceId:              serviceId,
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

// ServiceId returns the serviceId of the JSON-RPC service.
func (j *jsonRPCServer) ServiceId() *sharedtypes.ServiceId {
	return j.serviceId
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when jsonRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (j *jsonRPCServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	// Relay the request to the native service and build the response that will be sent back to the client.
	relay, err := j.serveHTTP(ctx, request)
	if err != nil {
		// Reply with an error if relay response could not be built.
		j.replyWithError(writer, err)
		return
	}

	// Send the relay response to the client.
	if err := j.sendRelayResponse(relay.Res, writer); err != nil {
		j.replyWithError(writer, err)
		return
	}

	// Emit the relay to the servedRelays observable.
	j.servedRelaysProducer <- relay
}

// serveHTTP holds the underlying logic of ServeHTTP.
func (j *jsonRPCServer) serveHTTP(ctx context.Context, request *http.Request) (*types.Relay, error) {
	// Extract the relay request from the request body.
	relayRequest, err := j.newRelayRequest(request)
	if err != nil {
		return nil, err
	}

	// Verify the relay request signature and session.
	if err := j.relayerProxy.VerifyRelayRequest(ctx, relayRequest, j.serviceId); err != nil {
		return nil, err
	}

	// Get the relayRequest payload's ReadCloser to add it to the http.Request
	// that will be sent to the native service.
	var payloadBz []byte
	if _, err = relayRequest.Payload.MarshalTo(payloadBz); err != nil {
		return nil, err
	}
	requestBodyReader := io.NopCloser(bytes.NewBuffer(payloadBz))

	// Build the request to be sent to the native service by substituting
	// the destination URL's host with the native service's listen address.
	destinationURL, err := url.Parse(request.URL.String())
	if err != nil {
		return nil, err
	}
	destinationURL.Host = j.nativeServiceListenAddress

	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    destinationURL,
		Host:   destinationURL.Host,
		Body:   requestBodyReader,
	}

	// Send the relay request to the native service.
	httpResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, err
	}

	// Build the relay response from the native service response
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it was verified to be valid
	// and has to be the same as the relayResponse session header.
	relayResponse, err := j.newRelayResponse(httpResponse, relayRequest.Meta.SessionHeader)
	if err != nil {
		return nil, err
	}

	return &types.Relay{Req: relayRequest, Res: relayResponse}, nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (j *jsonRPCServer) sendRelayResponse(relayResponse *types.RelayResponse, writer http.ResponseWriter) error {
	relayResposeBz, err := relayResponse.Marshal()
	if err != nil {
		return err
	}

	_, err = writer.Write(relayResposeBz)
	if err != nil {
		return err
	}

	return nil
}
