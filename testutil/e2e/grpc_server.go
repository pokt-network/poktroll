package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newGRPCServer creates and configures a new gRPC server for the E2EApp
func newGRPCServer(app *E2EApp, t *testing.T) *grpc.Server {
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	forwarder := &grpcForwarderServer{
		queryHelper: app.QueryHelper(),
		app:         app,
		t:           t,
	}

	grpcServer.RegisterService(&grpc.ServiceDesc{
		ServiceName: "cosmos.Service",
		HandlerType: (*interface{})(nil),
		Methods:     []grpc.MethodDesc{},
		Streams:     []grpc.StreamDesc{},
		Metadata:    "",
	}, forwarder)

	return grpcServer
}

// grpcForwarderServer implements a generic gRPC service that forwards all queries
// to the queryHelper and messages to the app
type grpcForwarderServer struct {
	queryHelper *baseapp.QueryServiceTestHelper
	app         *E2EApp
	t           *testing.T
}

// Invoke implements the grpc.Server interface and forwards all requests appropriately
func (s *grpcForwarderServer) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	// Determine if this is a query or message based on the method name
	if isQuery(method) {
		return s.queryHelper.Invoke(ctx, method, args, reply)
	}

	// If it's not a query, treat it as a message
	msg, ok := args.(sdk.Msg)
	if !ok {
		return fmt.Errorf("expected sdk.Msg, got %T", args)
	}

	// Run the message through the app
	msgRes, err := s.app.RunMsg(s.t, msg)
	if err != nil {
		return err
	}

	// Type assert the reply as a proto.Message
	protoReply, ok := reply.(proto.Message)
	if !ok {
		return fmt.Errorf("expected proto.Message, got %T", reply)
	}

	// Type assert the response as a proto.Message
	protoRes, ok := msgRes.(proto.Message)
	if !ok {
		return fmt.Errorf("expected proto.Message response, got %T", msgRes)
	}

	// Marshal the response to bytes
	resBz, err := proto.Marshal(protoRes)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Unmarshal into the reply
	return proto.Unmarshal(resBz, protoReply)
}

// NewStream implements the grpc.Server interface but is not used in this implementation
func (s *grpcForwarderServer) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, fmt.Errorf("streaming is not supported")
}

// isQuery returns true if the method name indicates this is a query request
func isQuery(method string) bool {
	return strings.Contains(method, ".Query/")
}
