package relayer

import (
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// GRPCWebWrapper wraps a gRPC server to handle gRPC-Web requests.
// gRPC-Web allows browser clients to communicate with gRPC services
// over HTTP/1.1 using the application/grpc-web content type.
type GRPCWebWrapper struct {
	logger        polylog.Logger
	wrappedServer *grpcweb.WrappedGrpcServer
}

// NewGRPCWebWrapper creates a new gRPC-Web wrapper around a gRPC server.
func NewGRPCWebWrapper(
	logger polylog.Logger,
	grpcServer *grpc.Server,
) *GRPCWebWrapper {
	// Create grpc-web wrapped server with options
	wrappedServer := grpcweb.WrapServer(grpcServer,
		// Allow all origins for development (configure appropriately for production)
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		// Allow credentials
		grpcweb.WithAllowNonRootResource(true),
		// Enable WebSocket transport for streaming
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
	)

	return &GRPCWebWrapper{
		logger:        logger.With(logging.FieldComponent, logging.ComponentGRPCBridge),
		wrappedServer: wrappedServer,
	}
}

// ServeHTTP handles HTTP requests that might be gRPC-Web requests.
// If the request is a gRPC-Web request, it's handled by the wrapped server.
// Otherwise, it returns false and the request should be handled by other handlers.
func (w *GRPCWebWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) bool {
	if w.wrappedServer.IsGrpcWebRequest(req) {
		w.logger.Debug().
			Str("method", req.Method).
			Str("path", req.URL.Path).
			Str("content-type", req.Header.Get("Content-Type")).
			Msg("handling gRPC-Web request")

		// Track gRPC-Web requests
		serviceID := req.Header.Get("Pocket-Service-Id")
		if serviceID != "" {
			grpcWebRequestsTotal.WithLabelValues(serviceID).Inc()
		}

		w.wrappedServer.ServeHTTP(resp, req)
		return true
	}
	return false
}

// IsGRPCWebRequest checks if the given request is a gRPC-Web request.
func (w *GRPCWebWrapper) IsGRPCWebRequest(req *http.Request) bool {
	return w.wrappedServer.IsGrpcWebRequest(req)
}

// IsAcceptableGRPCCorsRequest checks if the request is an acceptable
// CORS preflight request for gRPC-Web.
func (w *GRPCWebWrapper) IsAcceptableGRPCCorsRequest(req *http.Request) bool {
	return w.wrappedServer.IsAcceptableGrpcCorsRequest(req)
}
