package relayer

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/siderolabs/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// grpcMessageDirection represents the direction of a gRPC message.
type grpcMessageDirection string

const (
	grpcMessageDirectionClientToBackend grpcMessageDirection = "client_to_backend"
	grpcMessageDirectionBackendToClient grpcMessageDirection = "backend_to_client"
)

// grpcBackendConn represents a connection to a backend gRPC server.
type grpcBackendConn struct {
	conn    *grpc.ClientConn
	address string
	useTLS  bool
}

// billingBackend wraps a backend connection with billing capabilities.
// It implements the proxy.Backend interface.
type billingBackend struct {
	conn            *grpc.ClientConn
	handler         *GRPCProxyHandler
	serviceID       string
	supplierAddress string
	arrivalHeight   int64
}

// String returns the backend name for logging.
func (b *billingBackend) String() string {
	return b.serviceID
}

// GetConnection returns the gRPC connection to the backend.
func (b *billingBackend) GetConnection(ctx context.Context, _ string) (context.Context, *grpc.ClientConn, error) {
	// Forward metadata to the outgoing context
	md, _ := metadata.FromIncomingContext(ctx)
	outCtx := metadata.NewOutgoingContext(ctx, md)
	return outCtx, b.conn, nil
}

// AppendInfo is called to enhance response from backend with additional data.
// For simple one-to-one proxying, we just return the response unchanged.
func (b *billingBackend) AppendInfo(_ bool, resp []byte) ([]byte, error) {
	// Track backend-to-client message
	grpcMessagesForwarded.WithLabelValues(b.serviceID, string(grpcMessageDirectionBackendToClient)).Inc()
	return resp, nil
}

// BuildError converts errors from upstream into response fields.
// For one-to-one proxying, this is not called.
func (b *billingBackend) BuildError(_ bool, _ error) ([]byte, error) {
	return nil, nil
}

// GRPCProxyHandler manages the gRPC proxy server for the HA RelayMiner.
type GRPCProxyHandler struct {
	logger          polylog.Logger
	relayProcessor  RelayProcessor
	publisher       transport.MinedRelayPublisher
	responseSigner  *ResponseSigner
	supplierAddress string

	// Service configurations keyed by service ID
	serviceConfigs map[string]ServiceConfig

	// Backend connections keyed by service ID
	backends sync.Map // map[string]*grpcBackendConn

	// Block height for arrival tracking
	currentBlockHeight *atomic.Int64

	// gRPC server for handling proxy requests
	grpcServer *grpc.Server
}

// NewGRPCProxyHandler creates a new gRPC proxy handler.
func NewGRPCProxyHandler(
	logger polylog.Logger,
	serviceConfigs map[string]ServiceConfig,
	supplierAddress string,
	relayProcessor RelayProcessor,
	publisher transport.MinedRelayPublisher,
	responseSigner *ResponseSigner,
	currentBlockHeight *atomic.Int64,
) *GRPCProxyHandler {
	h := &GRPCProxyHandler{
		logger:             logger.With(logging.FieldComponent, logging.ComponentGRPCBridge),
		serviceConfigs:     serviceConfigs,
		supplierAddress:    supplierAddress,
		relayProcessor:     relayProcessor,
		publisher:          publisher,
		responseSigner:     responseSigner,
		currentBlockHeight: currentBlockHeight,
	}

	// Create gRPC server with the proxy handler
	h.grpcServer = grpc.NewServer(
		grpc.ForceServerCodecV2(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(h.director)),
		grpc.ChainStreamInterceptor(h.billingInterceptor()),
	)

	return h
}

// connectToBackend establishes a gRPC connection to the backend server.
func (h *GRPCProxyHandler) connectToBackend(serviceID string, backendURL string) (*grpcBackendConn, error) {
	// Check if we already have a connection
	if existing, ok := h.backends.Load(serviceID); ok {
		return existing.(*grpcBackendConn), nil
	}

	// Parse the backend URL to determine TLS settings
	useTLS := strings.HasPrefix(backendURL, "grpcs://") || strings.HasPrefix(backendURL, "https://")

	// Strip the scheme
	address := backendURL
	address = strings.TrimPrefix(address, "grpcs://")
	address = strings.TrimPrefix(address, "grpc://")
	address = strings.TrimPrefix(address, "https://")
	address = strings.TrimPrefix(address, "http://")

	// Build dial options
	var opts []grpc.DialOption
	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Use the proxy codec to forward messages without unmarshaling
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.ForceCodecV2(proxy.Codec())))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, err
	}

	backend := &grpcBackendConn{
		conn:    conn,
		address: address,
		useTLS:  useTLS,
	}

	// Store the connection
	h.backends.Store(serviceID, backend)

	h.logger.Info().
		Str(logging.FieldServiceID, serviceID).
		Str("backend_address", address).
		Bool("tls", useTLS).
		Msg("connected to gRPC backend")

	return backend, nil
}

// director implements proxy.StreamDirector to route gRPC calls to backends.
func (h *GRPCProxyHandler) director(ctx context.Context, fullMethodName string) (proxy.Mode, []proxy.Backend, error) {
	// Extract service ID from incoming metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return proxy.One2One, nil, status.Error(codes.InvalidArgument, "missing metadata")
	}

	serviceID := ""
	if vals := md.Get("pocket-service-id"); len(vals) > 0 {
		serviceID = vals[0]
	}

	if serviceID == "" {
		return proxy.One2One, nil, status.Error(codes.InvalidArgument, "missing pocket-service-id header")
	}

	// Get service configuration
	svcConfig, ok := h.serviceConfigs[serviceID]
	if !ok {
		return proxy.One2One, nil, status.Errorf(codes.NotFound, "unknown service: %s", serviceID)
	}

	// Get gRPC backend URL
	var backendURL string
	if backend, ok := svcConfig.Backends["grpc"]; ok {
		backendURL = backend.URL
	} else {
		return proxy.One2One, nil, status.Errorf(codes.Unavailable, "gRPC backend not configured for service: %s", serviceID)
	}

	// Connect to backend
	backendConn, err := h.connectToBackend(serviceID, backendURL)
	if err != nil {
		return proxy.One2One, nil, status.Errorf(codes.Unavailable, "failed to connect to backend: %v", err)
	}

	// Extract supplier address
	supplierAddress := h.supplierAddress
	if vals := md.Get("pocket-supplier-address"); len(vals) > 0 {
		supplierAddress = vals[0]
	}

	// Track stream metrics
	grpcStreamsActive.WithLabelValues(serviceID).Inc()
	grpcStreamsTotal.WithLabelValues(serviceID).Inc()

	// Create billing backend wrapper
	backend := &billingBackend{
		conn:            backendConn.conn,
		handler:         h,
		serviceID:       serviceID,
		supplierAddress: supplierAddress,
		arrivalHeight:   h.currentBlockHeight.Load(),
	}

	h.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str("method", fullMethodName).
		Msg("routing gRPC request to backend")

	return proxy.One2One, []proxy.Backend{backend}, nil
}

// billingInterceptor returns a gRPC stream server interceptor that tracks
// messages for billing purposes.
func (h *GRPCProxyHandler) billingInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract service ID from context
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return handler(srv, ss)
		}

		serviceID := ""
		if vals := md.Get("pocket-service-id"); len(vals) > 0 {
			serviceID = vals[0]
		}

		supplierAddress := h.supplierAddress
		if vals := md.Get("pocket-supplier-address"); len(vals) > 0 {
			supplierAddress = vals[0]
		}

		arrivalHeight := h.currentBlockHeight.Load()

		// Create a wrapped stream that intercepts messages for billing
		wrapped := &billingServerStream{
			ServerStream:    ss,
			handler:         h,
			serviceID:       serviceID,
			supplierAddress: supplierAddress,
			arrivalHeight:   arrivalHeight,
			method:          info.FullMethod,
		}

		// Run the handler with the wrapped stream
		err := handler(srv, wrapped)

		// Decrement active streams
		if serviceID != "" {
			grpcStreamsActive.WithLabelValues(serviceID).Dec()
		}

		// Log final stats
		if wrapped.messageCount.Load() > 0 {
			h.logger.Debug().
				Str(logging.FieldServiceID, serviceID).
				Str("method", info.FullMethod).
				Uint64("messages_billed", wrapped.messageCount.Load()).
				Msg("gRPC stream completed with billing")
		}

		return err
	}
}

// billingServerStream wraps a grpc.ServerStream to intercept messages for billing.
type billingServerStream struct {
	grpc.ServerStream
	handler         *GRPCProxyHandler
	serviceID       string
	supplierAddress string
	arrivalHeight   int64
	method          string
	messageCount    atomic.Uint64

	// Track request/response pairing
	lastRequestData []byte
	mu              sync.Mutex
}

// RecvMsg intercepts incoming messages (from client).
func (s *billingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	// Track the message for billing
	if s.serviceID != "" {
		grpcMessagesForwarded.WithLabelValues(s.serviceID, string(grpcMessageDirectionClientToBackend)).Inc()
	}

	return nil
}

// SendMsg intercepts outgoing messages (to client).
func (s *billingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		return err
	}

	// Track the message for billing
	if s.serviceID != "" {
		grpcMessagesForwarded.WithLabelValues(s.serviceID, string(grpcMessageDirectionBackendToClient)).Inc()
	}

	// Emit relay for this message
	s.emitRelay()

	return nil
}

// emitRelay creates and publishes a mined relay for a message.
func (s *billingServerStream) emitRelay() {
	if s.handler.publisher == nil {
		return
	}

	count := s.messageCount.Add(1)

	// Create basic relay message for gRPC
	// Note: For gRPC, we don't have access to raw request/response bytes in the interceptor
	// The billing backend's AppendInfo handles response tracking
	msg := &transport.MinedRelayMessage{
		RelayHash:               nil,
		RelayBytes:              nil, // gRPC messages handled by proxy
		ComputeUnitsPerRelay:    1,
		SupplierOperatorAddress: s.supplierAddress,
		ServiceId:               s.serviceID,
		ArrivalBlockHeight:      s.arrivalHeight,
	}
	msg.SetPublishedAt()

	if pubErr := s.handler.publisher.Publish(s.ServerStream.Context(), msg); pubErr != nil {
		s.handler.logger.Warn().Err(pubErr).Msg("failed to publish gRPC relay")
		return
	}

	grpcRelaysEmitted.WithLabelValues(s.serviceID).Inc()
	s.handler.logger.Debug().
		Uint64("relay_count", count).
		Str(logging.FieldSupplier, s.supplierAddress).
		Str("method", s.method).
		Msg("gRPC relay published")
}

// GetGRPCServer returns the underlying gRPC server for integration.
func (h *GRPCProxyHandler) GetGRPCServer() *grpc.Server {
	return h.grpcServer
}

// IsGRPCRequest checks if the request is a native gRPC request (HTTP/2 with
// application/grpc content type).
func IsGRPCRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/grpc")
}

// IsGRPCWebRequest checks if the request is a gRPC-Web request.
// gRPC-Web uses application/grpc-web or application/grpc-web+proto content types.
func IsGRPCWebRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/grpc-web")
}

// Close closes all backend connections and stops the gRPC server.
func (h *GRPCProxyHandler) Close() error {
	// Stop gRPC server gracefully
	if h.grpcServer != nil {
		h.grpcServer.GracefulStop()
	}

	// Close all backend connections
	var lastErr error
	h.backends.Range(func(key, value interface{}) bool {
		backend := value.(*grpcBackendConn)
		if err := backend.conn.Close(); err != nil {
			h.logger.Warn().
				Err(err).
				Str(logging.FieldServiceID, key.(string)).
				Msg("failed to close gRPC backend connection")
			lastErr = err
		}
		return true
	})

	return lastErr
}
