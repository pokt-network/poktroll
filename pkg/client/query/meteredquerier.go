package query

import (
	"context"
	"time"

	"github.com/cosmos/gogoproto/grpc"
	"github.com/pokt-network/poktroll/pkg/relayer"
	googlegrpc "google.golang.org/grpc"
)

// ComponentCtxRelayMiner is a type used to identify the component making the gRPC call.
type ComponentCtxRelayMiner string

// ComponentCtxRelayMinerKey is the context key used to identify the component calling the gRPC method
// using the context.WithValue method.
const ComponentCtxRelayMinerKey ComponentCtxRelayMiner = "component"

const (
	// ComponentCtxRelayMinerProxy is the identifier for the proxy component in the relayer.
	ComponentCtxRelayMinerProxy = iota
	// ComponentCtxRelayMinerMiner is the identifier for the miner component in the relayer.
	ComponentCtxRelayMinerMiner
	// ComponentCtxRelayMinerSessionsManager is the identifier for the sessions manager component in the relayer.
	ComponentCtxRelayMinerSessionsManager
)

// componentKindNames maps component context keys to their string representations.
// This is used to provide human-readable names for the components in metrics and logging.
var componentKindNames = map[int]string{
	ComponentCtxRelayMinerProxy:           "proxy",
	ComponentCtxRelayMinerMiner:           "miner",
	ComponentCtxRelayMinerSessionsManager: "sessions_manager",
}

// grpcClientWithDebugMetrics is a wrapper around grpc.ClientConn that captures the duration of gRPC calls.
// It implements the grpc.ClientConn interface and is used to monitor the performance of gRPC calls
// by recording the time taken for each call.
type grpcClientWithDebugMetrics struct {
	grpc.ClientConn
}

// NewGRPCClientWithDebugMetrics creates a new grpcClientWithDebugMetrics that wraps the provided grpc.ClientConn.
// It is used to instrument gRPC calls for performance monitoring.
func NewGRPCClientWithDebugMetrics(clientConn grpc.ClientConn) grpc.ClientConn {
	return &grpcClientWithDebugMetrics{
		ClientConn: clientConn,
	}
}

// Invoke wraps the ClientConn's Invoke method to capture the duration of the call.
//   - It uses the gRPC method name as the method being invoked.
//   - It uses the context to retrieve the component kind (e.g., ComponentCtxProxy...)
//     which is used to differentiate between different types of callers.
func (m *grpcClientWithDebugMetrics) Invoke(ctx context.Context, method string, args, reply any, opts ...googlegrpc.CallOption) error {
	now := time.Now()
	component := ctx.Value(ComponentCtxRelayMinerKey)
	defer relayer.CaptureGRPCCallDuration(componentKindNames[component.(int)], method, now)

	return m.ClientConn.Invoke(ctx, method, args, reply, opts...)
}
