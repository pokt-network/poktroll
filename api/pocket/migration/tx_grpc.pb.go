// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             (unknown)
// source: pocket/migration/tx.proto

package migration

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
	Msg_UpdateParams_FullMethodName                 = "/pocket.migration.Msg/UpdateParams"
	Msg_ImportMorseClaimableAccounts_FullMethodName = "/pocket.migration.Msg/ImportMorseClaimableAccounts"
	Msg_ClaimMorseAccount_FullMethodName            = "/pocket.migration.Msg/ClaimMorseAccount"
	Msg_ClaimMorseApplication_FullMethodName        = "/pocket.migration.Msg/ClaimMorseApplication"
	Msg_ClaimMorseSupplier_FullMethodName           = "/pocket.migration.Msg/ClaimMorseSupplier"
	Msg_RecoverMorseAccount_FullMethodName          = "/pocket.migration.Msg/RecoverMorseAccount"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Msg defines the Msg service.
//
// - Provides RPCs for migration-related operations
// - Includes parameter updates, Morse account claims, supplier claims, and recovery
type MsgClient interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	ImportMorseClaimableAccounts(ctx context.Context, in *MsgImportMorseClaimableAccounts, opts ...grpc.CallOption) (*MsgImportMorseClaimableAccountsResponse, error)
	ClaimMorseAccount(ctx context.Context, in *MsgClaimMorseAccount, opts ...grpc.CallOption) (*MsgClaimMorseAccountResponse, error)
	ClaimMorseApplication(ctx context.Context, in *MsgClaimMorseApplication, opts ...grpc.CallOption) (*MsgClaimMorseApplicationResponse, error)
	ClaimMorseSupplier(ctx context.Context, in *MsgClaimMorseSupplier, opts ...grpc.CallOption) (*MsgClaimMorseSupplierResponse, error)
	RecoverMorseAccount(ctx context.Context, in *MsgRecoverMorseAccount, opts ...grpc.CallOption) (*MsgRecoverMorseAccountResponse, error)
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

func (c *msgClient) ImportMorseClaimableAccounts(ctx context.Context, in *MsgImportMorseClaimableAccounts, opts ...grpc.CallOption) (*MsgImportMorseClaimableAccountsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgImportMorseClaimableAccountsResponse)
	err := c.cc.Invoke(ctx, Msg_ImportMorseClaimableAccounts_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ClaimMorseAccount(ctx context.Context, in *MsgClaimMorseAccount, opts ...grpc.CallOption) (*MsgClaimMorseAccountResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgClaimMorseAccountResponse)
	err := c.cc.Invoke(ctx, Msg_ClaimMorseAccount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ClaimMorseApplication(ctx context.Context, in *MsgClaimMorseApplication, opts ...grpc.CallOption) (*MsgClaimMorseApplicationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgClaimMorseApplicationResponse)
	err := c.cc.Invoke(ctx, Msg_ClaimMorseApplication_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ClaimMorseSupplier(ctx context.Context, in *MsgClaimMorseSupplier, opts ...grpc.CallOption) (*MsgClaimMorseSupplierResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgClaimMorseSupplierResponse)
	err := c.cc.Invoke(ctx, Msg_ClaimMorseSupplier_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) RecoverMorseAccount(ctx context.Context, in *MsgRecoverMorseAccount, opts ...grpc.CallOption) (*MsgRecoverMorseAccountResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgRecoverMorseAccountResponse)
	err := c.cc.Invoke(ctx, Msg_RecoverMorseAccount_FullMethodName, in, out, cOpts...)
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
//
// - Provides RPCs for migration-related operations
// - Includes parameter updates, Morse account claims, supplier claims, and recovery
type MsgServer interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	ImportMorseClaimableAccounts(context.Context, *MsgImportMorseClaimableAccounts) (*MsgImportMorseClaimableAccountsResponse, error)
	ClaimMorseAccount(context.Context, *MsgClaimMorseAccount) (*MsgClaimMorseAccountResponse, error)
	ClaimMorseApplication(context.Context, *MsgClaimMorseApplication) (*MsgClaimMorseApplicationResponse, error)
	ClaimMorseSupplier(context.Context, *MsgClaimMorseSupplier) (*MsgClaimMorseSupplierResponse, error)
	RecoverMorseAccount(context.Context, *MsgRecoverMorseAccount) (*MsgRecoverMorseAccountResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (UnimplementedMsgServer) ImportMorseClaimableAccounts(context.Context, *MsgImportMorseClaimableAccounts) (*MsgImportMorseClaimableAccountsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ImportMorseClaimableAccounts not implemented")
}
func (UnimplementedMsgServer) ClaimMorseAccount(context.Context, *MsgClaimMorseAccount) (*MsgClaimMorseAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClaimMorseAccount not implemented")
}
func (UnimplementedMsgServer) ClaimMorseApplication(context.Context, *MsgClaimMorseApplication) (*MsgClaimMorseApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClaimMorseApplication not implemented")
}
func (UnimplementedMsgServer) ClaimMorseSupplier(context.Context, *MsgClaimMorseSupplier) (*MsgClaimMorseSupplierResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClaimMorseSupplier not implemented")
}
func (UnimplementedMsgServer) RecoverMorseAccount(context.Context, *MsgRecoverMorseAccount) (*MsgRecoverMorseAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecoverMorseAccount not implemented")
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

func _Msg_ImportMorseClaimableAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgImportMorseClaimableAccounts)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ImportMorseClaimableAccounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ImportMorseClaimableAccounts_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ImportMorseClaimableAccounts(ctx, req.(*MsgImportMorseClaimableAccounts))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ClaimMorseAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimMorseAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ClaimMorseAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ClaimMorseAccount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ClaimMorseAccount(ctx, req.(*MsgClaimMorseAccount))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ClaimMorseApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimMorseApplication)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ClaimMorseApplication(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ClaimMorseApplication_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ClaimMorseApplication(ctx, req.(*MsgClaimMorseApplication))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ClaimMorseSupplier_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimMorseSupplier)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ClaimMorseSupplier(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ClaimMorseSupplier_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ClaimMorseSupplier(ctx, req.(*MsgClaimMorseSupplier))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_RecoverMorseAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRecoverMorseAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RecoverMorseAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_RecoverMorseAccount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RecoverMorseAccount(ctx, req.(*MsgRecoverMorseAccount))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pocket.migration.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "ImportMorseClaimableAccounts",
			Handler:    _Msg_ImportMorseClaimableAccounts_Handler,
		},
		{
			MethodName: "ClaimMorseAccount",
			Handler:    _Msg_ClaimMorseAccount_Handler,
		},
		{
			MethodName: "ClaimMorseApplication",
			Handler:    _Msg_ClaimMorseApplication_Handler,
		},
		{
			MethodName: "ClaimMorseSupplier",
			Handler:    _Msg_ClaimMorseSupplier_Handler,
		},
		{
			MethodName: "RecoverMorseAccount",
			Handler:    _Msg_RecoverMorseAccount_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pocket/migration/tx.proto",
}
