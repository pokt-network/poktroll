package relayer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// RelayServiceMethodPath is the gRPC method path for the relay service.
// Clients (e.g., PATH gateway) call this method with a RelayRequest message.
const RelayServiceMethodPath = "/pocket.service.RelayService/SendRelay"

// RelayGRPCService implements a gRPC service that properly handles the relay protocol.
// It receives RelayRequest messages, extracts metadata, forwards to backends, and
// returns signed RelayResponse messages.
type RelayGRPCService struct {
	logger         polylog.Logger
	serviceConfigs map[string]ServiceConfig
	responseSigner *ResponseSigner
	publisher      transport.MinedRelayPublisher
	relayProcessor RelayProcessor
	httpClient     *http.Client

	// Backend gRPC connections for passthrough mode
	grpcBackends sync.Map // map[string]*grpc.ClientConn

	// Block height tracking
	currentBlockHeight *atomic.Int64

	// Max response body size
	maxBodySize int64
}

// RelayGRPCServiceConfig contains configuration for the relay gRPC service.
type RelayGRPCServiceConfig struct {
	ServiceConfigs     map[string]ServiceConfig
	ResponseSigner     *ResponseSigner
	Publisher          transport.MinedRelayPublisher
	RelayProcessor     RelayProcessor
	CurrentBlockHeight *atomic.Int64
	MaxBodySize        int64
	HTTPClient         *http.Client
}

// NewRelayGRPCService creates a new gRPC relay service.
func NewRelayGRPCService(logger polylog.Logger, config RelayGRPCServiceConfig) *RelayGRPCService {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		}
	}

	maxBodySize := config.MaxBodySize
	if maxBodySize == 0 {
		maxBodySize = 10 * 1024 * 1024 // 10MB default
	}

	return &RelayGRPCService{
		logger:             logger.With(logging.FieldComponent, "grpc_relay_service"),
		serviceConfigs:     config.ServiceConfigs,
		responseSigner:     config.ResponseSigner,
		publisher:          config.Publisher,
		relayProcessor:     config.RelayProcessor,
		currentBlockHeight: config.CurrentBlockHeight,
		maxBodySize:        maxBodySize,
		httpClient:         httpClient,
	}
}

// RegisterWithServer registers the relay service handler with a gRPC server.
// This uses the UnknownServiceHandler pattern to intercept calls to our method path.
func (s *RelayGRPCService) RegisterWithServer(server *grpc.Server) {
	// Note: We use UnknownServiceHandler in the server options instead of registering here.
	// This is because we're handling a dynamically defined service.
	s.logger.Info().Msg("relay gRPC service registered")
}

// HandleUnknownService is a gRPC stream handler that processes relay requests.
// It should be registered as the UnknownServiceHandler on the gRPC server.
func (s *RelayGRPCService) HandleUnknownService(srv interface{}, stream grpc.ServerStream) error {
	// Get the full method name from the stream
	fullMethod, ok := grpc.Method(stream.Context())
	if !ok {
		return status.Error(codes.Internal, "failed to get method name")
	}

	// Check if this is a relay request
	if fullMethod != RelayServiceMethodPath {
		return status.Errorf(codes.Unimplemented, "unknown method: %s", fullMethod)
	}

	return s.handleSendRelay(stream)
}

// handleSendRelay processes a SendRelay gRPC call.
func (s *RelayGRPCService) handleSendRelay(stream grpc.ServerStream) error {
	ctx := stream.Context()
	arrivalTime := time.Now()
	arrivalHeight := int64(0)
	if s.currentBlockHeight != nil {
		arrivalHeight = s.currentBlockHeight.Load()
	}

	// Receive the RelayRequest message (typed proto message)
	relayRequest := &servicetypes.RelayRequest{}
	if err := stream.RecvMsg(relayRequest); err != nil {
		grpcRelayErrors.WithLabelValues("unknown", "recv_error").Inc()
		return status.Errorf(codes.InvalidArgument, "failed to receive request: %v", err)
	}

	// Extract metadata from RelayRequest.Meta
	if relayRequest.Meta.SessionHeader == nil {
		grpcRelayErrors.WithLabelValues("unknown", "missing_session_header").Inc()
		return status.Error(codes.InvalidArgument, "missing session header in RelayRequest")
	}

	serviceID := relayRequest.Meta.SessionHeader.ServiceId
	supplierOperatorAddr := relayRequest.Meta.SupplierOperatorAddress
	applicationAddr := relayRequest.Meta.SessionHeader.ApplicationAddress
	sessionID := relayRequest.Meta.SessionHeader.SessionId

	if serviceID == "" {
		grpcRelayErrors.WithLabelValues("unknown", "missing_service_id").Inc()
		return status.Error(codes.InvalidArgument, "missing service ID in session header")
	}

	if supplierOperatorAddr == "" {
		grpcRelayErrors.WithLabelValues(serviceID, "missing_supplier_address").Inc()
		return status.Error(codes.InvalidArgument, "missing supplier operator address in RelayRequest")
	}

	s.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str(logging.FieldSupplier, supplierOperatorAddr).
		Str("application", applicationAddr).
		Str("session_id", sessionID).
		Msg("received gRPC relay request")

	// Verify we have a signer for this supplier
	if s.responseSigner == nil || !s.responseSigner.HasSigner(supplierOperatorAddr) {
		grpcRelayErrors.WithLabelValues(serviceID, "no_signer").Inc()
		return status.Errorf(codes.FailedPrecondition, "no signer for supplier %s", supplierOperatorAddr)
	}

	// Get service configuration
	svcConfig, ok := s.serviceConfigs[serviceID]
	if !ok {
		grpcRelayErrors.WithLabelValues(serviceID, "unknown_service").Inc()
		return status.Errorf(codes.NotFound, "unknown service: %s", serviceID)
	}

	// Deserialize the POKTHTTPRequest from the relay payload
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		grpcRelayErrors.WithLabelValues(serviceID, "payload_deserialize_error").Inc()
		return status.Errorf(codes.InvalidArgument, "failed to deserialize POKTHTTPRequest: %v", err)
	}

	s.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str("method", poktHTTPRequest.Method).
		Str("url", poktHTTPRequest.Url).
		Int("body_size", len(poktHTTPRequest.BodyBz)).
		Msg("deserialized POKTHTTPRequest from relay payload")

	// Forward request to backend and get response
	respBody, respHeaders, respStatus, err := s.forwardToBackend(ctx, serviceID, &svcConfig, poktHTTPRequest)
	if err != nil {
		grpcRelayErrors.WithLabelValues(serviceID, "backend_error").Inc()
		s.logger.Error().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("failed to forward request to backend")

		// Build error response
		relayResponse, _, buildErr := s.responseSigner.BuildErrorRelayResponse(
			relayRequest.Meta.SessionHeader,
			supplierOperatorAddr,
			500,
			fmt.Sprintf("backend error: %v", err),
		)
		if buildErr != nil {
			return status.Errorf(codes.Internal, "failed to build error response: %v", buildErr)
		}

		// Send error response (typed proto message)
		if sendErr := stream.SendMsg(relayResponse); sendErr != nil {
			return status.Errorf(codes.Internal, "failed to send error response: %v", sendErr)
		}

		// Still publish the relay for tracking (even failed ones)
		s.publishRelay(ctx, relayRequest, relayResponse, arrivalHeight, serviceID)

		return nil
	}

	// Build and sign the RelayResponse
	relayResponse, relayResponseBz, err := s.responseSigner.BuildAndSignRelayResponseFromBody(
		relayRequest,
		respBody,
		respHeaders,
		respStatus,
	)
	if err != nil {
		grpcRelayErrors.WithLabelValues(serviceID, "sign_error").Inc()
		return status.Errorf(codes.Internal, "failed to build/sign response: %v", err)
	}

	// Send the response (typed proto message)
	if err := stream.SendMsg(relayResponse); err != nil {
		grpcRelayErrors.WithLabelValues(serviceID, "send_error").Inc()
		return status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

	// Publish the mined relay
	s.publishRelay(ctx, relayRequest, relayResponse, arrivalHeight, serviceID)

	// Update metrics
	grpcRelaysTotal.WithLabelValues(serviceID).Inc()
	grpcRelayLatency.WithLabelValues(serviceID).Observe(time.Since(arrivalTime).Seconds())

	s.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str(logging.FieldSupplier, supplierOperatorAddr).
		Int("response_size", len(relayResponseBz)).
		Dur("latency", time.Since(arrivalTime)).
		Msg("gRPC relay completed successfully")

	return nil
}

// forwardToBackend forwards the request to the appropriate backend service.
func (s *RelayGRPCService) forwardToBackend(
	ctx context.Context,
	serviceID string,
	svcConfig *ServiceConfig,
	poktHTTPRequest *sdktypes.POKTHTTPRequest,
) ([]byte, http.Header, int, error) {
	// Determine RPC type from content-type or use default
	rpcType := "rest"
	if poktHTTPRequest.Header != nil {
		if ctHeader, ok := poktHTTPRequest.Header["Content-Type"]; ok && len(ctHeader.Values) > 0 {
			contentType := ctHeader.Values[0]
			if strings.HasPrefix(contentType, "application/grpc") {
				rpcType = "grpc"
			}
		}
	}

	// Find the backend configuration
	var backendURL string
	var configHeaders map[string]string
	var auth *AuthenticationConfig

	if backend, ok := svcConfig.Backends[rpcType]; ok {
		backendURL = backend.URL
		configHeaders = backend.Headers
		auth = backend.Authentication
	} else if backend, ok := svcConfig.Backends["rest"]; ok {
		backendURL = backend.URL
		configHeaders = backend.Headers
		auth = backend.Authentication
	} else {
		// Use any available backend
		for _, backend := range svcConfig.Backends {
			backendURL = backend.URL
			configHeaders = backend.Headers
			auth = backend.Authentication
			break
		}
	}

	if backendURL == "" {
		return nil, nil, 0, fmt.Errorf("no backend configured for service %s", serviceID)
	}

	// Build the request URL
	requestURL, err := url.Parse(poktHTTPRequest.Url)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse request URL: %w", err)
	}

	backendParsed, err := url.Parse(backendURL)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse backend URL: %w", err)
	}

	// Merge URLs
	requestURL.Scheme = backendParsed.Scheme
	requestURL.Host = backendParsed.Host
	if backendParsed.Path != "" && backendParsed.Path != "/" {
		requestURL.Path = backendParsed.Path + requestURL.Path
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, poktHTTPRequest.Method, requestURL.String(), bytes.NewReader(poktHTTPRequest.BodyBz))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers from POKTHTTPRequest
	if poktHTTPRequest.Header != nil {
		for key, header := range poktHTTPRequest.Header {
			for _, value := range header.Values {
				req.Header.Add(key, value)
			}
		}
	}

	// Add config headers (override any matching keys)
	for key, value := range configHeaders {
		req.Header.Set(key, value)
	}

	// Add authentication if configured
	if auth != nil && auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}

	// Set host header
	req.Host = backendParsed.Host

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, s.maxBodySize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(respBody)) > s.maxBodySize {
		return nil, nil, 0, fmt.Errorf("response too large: %d > %d", len(respBody), s.maxBodySize)
	}

	return respBody, resp.Header, resp.StatusCode, nil
}

// publishRelay publishes the mined relay to the transport layer.
func (s *RelayGRPCService) publishRelay(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
	relayResponse *servicetypes.RelayResponse,
	arrivalHeight int64,
	serviceID string,
) {
	if s.publisher == nil {
		return
	}

	supplierOperatorAddr := relayRequest.Meta.SupplierOperatorAddress

	// Build the relay for mining
	relay := &servicetypes.Relay{
		Req: relayRequest,
		Res: relayResponse,
	}

	relayBz, err := relay.Marshal()
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to marshal relay for publishing")
		return
	}

	// Compute relay hash (convert fixed-size array to slice)
	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	// Build mined relay message
	msg := &transport.MinedRelayMessage{
		RelayHash:               relayHash,
		RelayBytes:              relayBz,
		ComputeUnitsPerRelay:    1, // TODO: Get from service config
		SupplierOperatorAddress: supplierOperatorAddr,
		ServiceId:               serviceID,
		ArrivalBlockHeight:      arrivalHeight,
	}
	msg.SetPublishedAt()

	// Add session metadata if available
	if relayRequest.Meta.SessionHeader != nil {
		msg.SessionId = relayRequest.Meta.SessionHeader.SessionId
		msg.ApplicationAddress = relayRequest.Meta.SessionHeader.ApplicationAddress
		msg.SessionEndHeight = relayRequest.Meta.SessionHeader.SessionEndBlockHeight
	}

	// Publish
	if err := s.publisher.Publish(ctx, msg); err != nil {
		s.logger.Warn().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Str(logging.FieldSupplier, supplierOperatorAddr).
			Msg("failed to publish gRPC relay")
		return
	}

	grpcRelaysPublished.WithLabelValues(serviceID).Inc()

	s.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str(logging.FieldSupplier, supplierOperatorAddr).
		Msg("gRPC relay published")
}

// connectToGRPCBackend establishes a gRPC connection to a backend (for future streaming support).
func (s *RelayGRPCService) connectToGRPCBackend(backendURL string) (*grpc.ClientConn, error) {
	// Check cache
	if conn, ok := s.grpcBackends.Load(backendURL); ok {
		return conn.(*grpc.ClientConn), nil
	}

	// Parse URL to determine TLS
	useTLS := strings.HasPrefix(backendURL, "grpcs://") || strings.HasPrefix(backendURL, "https://")

	// Strip scheme
	address := backendURL
	address = strings.TrimPrefix(address, "grpcs://")
	address = strings.TrimPrefix(address, "grpc://")
	address = strings.TrimPrefix(address, "https://")
	address = strings.TrimPrefix(address, "http://")

	var opts []grpc.DialOption
	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, err
	}

	s.grpcBackends.Store(backendURL, conn)
	return conn, nil
}

// Close closes all backend connections.
func (s *RelayGRPCService) Close() error {
	var lastErr error
	s.grpcBackends.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*grpc.ClientConn); ok {
			if err := conn.Close(); err != nil {
				lastErr = err
			}
		}
		return true
	})
	return lastErr
}

// NewGRPCServerForRelayService creates a gRPC server configured for the relay service.
// It uses the standard proto codec and handles typed RelayRequest/RelayResponse messages.
func NewGRPCServerForRelayService(service *RelayGRPCService) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnknownServiceHandler(service.HandleUnknownService),
	)
	return server
}
