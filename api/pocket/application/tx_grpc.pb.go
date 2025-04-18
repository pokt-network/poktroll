// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             (unknown)
// source: pocket/application/tx.proto

package application

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	Msg_UpdateParams_FullMethodName          = "/pocket.application.Msg/UpdateParams"
	Msg_StakeApplication_FullMethodName      = "/pocket.application.Msg/StakeApplication"
	Msg_UnstakeApplication_FullMethodName    = "/pocket.application.Msg/UnstakeApplication"
	Msg_DelegateToGateway_FullMethodName     = "/pocket.application.Msg/DelegateToGateway"
	Msg_UndelegateFromGateway_FullMethodName = "/pocket.application.Msg/UndelegateFromGateway"
	Msg_TransferApplication_FullMethodName   = "/pocket.application.Msg/TransferApplication"
	Msg_UpdateParam_FullMethodName           = "/pocket.application.Msg/UpdateParam"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Msg defines the Msg service.
type MsgClient interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	StakeApplication(ctx context.Context, in *MsgStakeApplication, opts ...grpc.CallOption) (*MsgStakeApplicationResponse, error)
	UnstakeApplication(ctx context.Context, in *MsgUnstakeApplication, opts ...grpc.CallOption) (*MsgUnstakeApplicationResponse, error)
	DelegateToGateway(ctx context.Context, in *MsgDelegateToGateway, opts ...grpc.CallOption) (*MsgDelegateToGatewayResponse, error)
	UndelegateFromGateway(ctx context.Context, in *MsgUndelegateFromGateway, opts ...grpc.CallOption) (*MsgUndelegateFromGatewayResponse, error)
	TransferApplication(ctx context.Context, in *MsgTransferApplication, opts ...grpc.CallOption) (*MsgTransferApplicationResponse, error)
	UpdateParam(ctx context.Context, in *MsgUpdateParam, opts ...grpc.CallOption) (*MsgUpdateParamResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, Msg_UpdateParams_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) StakeApplication(ctx context.Context, in *MsgStakeApplication, opts ...grpc.CallOption) (*MsgStakeApplicationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgStakeApplicationResponse)
	err := c.cc.Invoke(ctx, Msg_StakeApplication_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UnstakeApplication(ctx context.Context, in *MsgUnstakeApplication, opts ...grpc.CallOption) (*MsgUnstakeApplicationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUnstakeApplicationResponse)
	err := c.cc.Invoke(ctx, Msg_UnstakeApplication_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) DelegateToGateway(ctx context.Context, in *MsgDelegateToGateway, opts ...grpc.CallOption) (*MsgDelegateToGatewayResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgDelegateToGatewayResponse)
	err := c.cc.Invoke(ctx, Msg_DelegateToGateway_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UndelegateFromGateway(ctx context.Context, in *MsgUndelegateFromGateway, opts ...grpc.CallOption) (*MsgUndelegateFromGatewayResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUndelegateFromGatewayResponse)
	err := c.cc.Invoke(ctx, Msg_UndelegateFromGateway_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) TransferApplication(ctx context.Context, in *MsgTransferApplication, opts ...grpc.CallOption) (*MsgTransferApplicationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgTransferApplicationResponse)
	err := c.cc.Invoke(ctx, Msg_TransferApplication_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateParam(ctx context.Context, in *MsgUpdateParam, opts ...grpc.CallOption) (*MsgUpdateParamResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUpdateParamResponse)
	err := c.cc.Invoke(ctx, Msg_UpdateParam_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
//
// Msg defines the Msg service.
type MsgServer interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	StakeApplication(context.Context, *MsgStakeApplication) (*MsgStakeApplicationResponse, error)
	UnstakeApplication(context.Context, *MsgUnstakeApplication) (*MsgUnstakeApplicationResponse, error)
	DelegateToGateway(context.Context, *MsgDelegateToGateway) (*MsgDelegateToGatewayResponse, error)
	UndelegateFromGateway(context.Context, *MsgUndelegateFromGateway) (*MsgUndelegateFromGatewayResponse, error)
	TransferApplication(context.Context, *MsgTransferApplication) (*MsgTransferApplicationResponse, error)
	UpdateParam(context.Context, *MsgUpdateParam) (*MsgUpdateParamResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (UnimplementedMsgServer) StakeApplication(context.Context, *MsgStakeApplication) (*MsgStakeApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StakeApplication not implemented")
}
func (UnimplementedMsgServer) UnstakeApplication(context.Context, *MsgUnstakeApplication) (*MsgUnstakeApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnstakeApplication not implemented")
}
func (UnimplementedMsgServer) DelegateToGateway(context.Context, *MsgDelegateToGateway) (*MsgDelegateToGatewayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DelegateToGateway not implemented")
}
func (UnimplementedMsgServer) UndelegateFromGateway(context.Context, *MsgUndelegateFromGateway) (*MsgUndelegateFromGatewayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UndelegateFromGateway not implemented")
}
func (UnimplementedMsgServer) TransferApplication(context.Context, *MsgTransferApplication) (*MsgTransferApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TransferApplication not implemented")
}
func (UnimplementedMsgServer) UpdateParam(context.Context, *MsgUpdateParam) (*MsgUpdateParamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParam not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UpdateParams_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_StakeApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStakeApplication)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StakeApplication(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_StakeApplication_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StakeApplication(ctx, req.(*MsgStakeApplication))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UnstakeApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUnstakeApplication)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UnstakeApplication(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UnstakeApplication_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UnstakeApplication(ctx, req.(*MsgUnstakeApplication))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_DelegateToGateway_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDelegateToGateway)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DelegateToGateway(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_DelegateToGateway_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DelegateToGateway(ctx, req.(*MsgDelegateToGateway))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UndelegateFromGateway_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUndelegateFromGateway)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UndelegateFromGateway(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UndelegateFromGateway_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UndelegateFromGateway(ctx, req.(*MsgUndelegateFromGateway))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_TransferApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgTransferApplication)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).TransferApplication(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_TransferApplication_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).TransferApplication(ctx, req.(*MsgTransferApplication))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UpdateParam_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParam)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParam(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UpdateParam_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParam(ctx, req.(*MsgUpdateParam))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pocket.application.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "StakeApplication",
			Handler:    _Msg_StakeApplication_Handler,
		},
		{
			MethodName: "UnstakeApplication",
			Handler:    _Msg_UnstakeApplication_Handler,
		},
		{
			MethodName: "DelegateToGateway",
			Handler:    _Msg_DelegateToGateway_Handler,
		},
		{
			MethodName: "UndelegateFromGateway",
			Handler:    _Msg_UndelegateFromGateway_Handler,
		},
		{
			MethodName: "TransferApplication",
			Handler:    _Msg_TransferApplication_Handler,
		},
		{
			MethodName: "UpdateParam",
			Handler:    _Msg_UpdateParam_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pocket/application/tx.proto",
}
