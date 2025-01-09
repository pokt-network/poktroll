package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// newGRPCServer creates and configures a new gRPC server for the E2EApp
func newGRPCServer(app *E2EApp, t *testing.T) *grpc.Server {
	grpcServer := grpc.NewServer()
	app.RegisterGRPCServer(grpcServer)

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

//func newGRPCServer(app *E2EApp, t *testing.T) *grpc.Server {
//	grpcServer := grpc.NewServer()
//	reflection.Register(grpcServer)
//
//	forwarder := &grpcForwarderServer{
//		app:         app,
//		t:           t,
//		queryHelper: app.QueryHelper(),
//		msgRouter:   app.MsgServiceRouter(),
//		msgHandlers: map[string]interface{}{},
//	}
//
//	// Forward all gRPC messages through our forwarder
//	sd := &grpc.ServiceDesc{
//		ServiceName: "cosmos.Service",
//		HandlerType: (*interface{})(nil),
//		Methods: []grpc.MethodDesc{
//			{
//				MethodName: "HandleMessage",
//				Handler:    forwarder.handleMessageGeneric,
//			},
//		},
//	}
//	grpcServer.RegisterService(sd, forwarder)
//
//	return grpcServer
//}
//
//type grpcForwarderServer struct {
//	app         *E2EApp
//	t           *testing.T
//	queryHelper *baseapp.QueryServiceTestHelper
//	msgRouter   *baseapp.MsgServiceRouter
//	msgHandlers map[string]interface{}
//}
//
//func (s *grpcForwarderServer) handleMessageGeneric(
//	srv interface{},
//	ctx context.Context,
//	dec func(interface{}) error,
//	interceptor grpc.UnaryServerInterceptor,
//) (interface{}, error) {
//	msg, ok := srv.(sdk.Msg)
//	if !ok {
//		return nil, fmt.Errorf("invalid message type: %T", srv)
//	}
//
//	// Use the app's existing message handling infrastructure
//	msgRes, err := s.app.RunMsg(s.t, msg)
//	if err != nil {
//		return nil, err
//	}
//
//	return msgRes, nil
//}

//func newGRPCServer(app *E2EApp, t *testing.T) *grpc.Server {
//	grpcServer := grpc.NewServer()
//	reflection.Register(grpcServer)
//
//	// Register a service handler that forwards to MsgServiceRouter
//	msgHandler := &grpcForwarderServer{app: app, t: t}
//	serverServiceDesc := &grpc.ServiceDesc{
//		ServiceName: "cosmos.msg.v1.Msg",
//		HandlerType: (*interface{})(nil),
//		Methods: []grpc.MethodDesc{{
//			MethodName: "HandleMessage",
//			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
//				var msg sdk.Msg
//				if err := dec(&msg); err != nil {
//					return nil, err
//				}
//				return msgHandler.app.RunMsg(msgHandler.t, msg)
//			},
//		}},
//	}
//	grpcServer.RegisterService(serverServiceDesc, msgHandler)
//
//	// Set up the gRPC-Gateway mux
//	mux := runtime.NewServeMux()
//	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
//
//	// Register all your service handlers with the mux
//	if err := gatewaytypes.RegisterMsgHandlerFromEndpoint(context.Background(), mux, app.grpcListener.Addr().String(), opts); err != nil {
//		panic(err)
//	}
//
//	// Start HTTP server with the mux
//	go func() {
//		if err := http.ListenAndServe(":42070", mux); err != nil {
//			panic(err)
//		}
//	}()
//
//	return grpcServer
//}
