package query

import (
	"context"
	"time"

	"github.com/cosmos/gogoproto/grpc"
	"github.com/pokt-network/poktroll/pkg/relayer"
	googlegrpc "google.golang.org/grpc"
)

const (
	// ComponentCtxKey is the context key used to identify the component calling the gRPC method
	// using the context.WithValue method.
	ComponentCtxKey = "component"
	// ComponentCtxProxy is the identifier for the proxy component in the relayer.
	ComponentCtxProxy = iota
	// ComponentCtxMiner is the identifier for the miner component in the relayer.
	ComponentCtxMiner
	// ComponentCtxSessionsManager is the identifier for the sessions manager component in the relayer.
	ComponentCtxSessionsManager
)

// componentKindNames maps component context keys to their string representations.
// This is used to provide human-readable names for the components in metrics and logging.
var componentKindNames = map[int]string{
	ComponentCtxProxy:           "proxy",
	ComponentCtxMiner:           "miner",
	ComponentCtxSessionsManager: "sessions_manager",
}

// meteredClientConn is a wrapper around grpc.ClientConn that captures the duration of gRPC calls.
// It implements the grpc.ClientConn interface and is used to monitor the performance of gRPC calls
// by recording the time taken for each call.
type meteredClientConn struct {
	grpc.ClientConn
}

// NewMeteredClientConn creates a new meteredClientConn that wraps the provided grpc.ClientConn.
// It is used to instrument gRPC calls for performance monitoring.
func NewMeteredClientConn(clientConn grpc.ClientConn) grpc.ClientConn {
	return &meteredClientConn{
		ClientConn: clientConn,
	}
}

// Invoke wraps the ClientConn's Invoke method to capture the duration of the call.
//   - It uses the gRPC method name as the method being invoked.
//   - It uses the context to retrieve the component kind (e.g., ComponentCtxProxy...)
//     which is used to differentiate between different types of callers.
func (m *meteredClientConn) Invoke(ctx context.Context, method string, args, reply any, opts ...googlegrpc.CallOption) error {
	now := time.Now()
	component := ctx.Value(ComponentCtxKey)
	defer relayer.CapturePocketGRPCCallDuration(componentKindNames[component.(int)], method, now)

	return m.ClientConn.Invoke(ctx, method, args, reply, opts...)
}
