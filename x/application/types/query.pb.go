// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: poktroll/application/query.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// QueryParamsRequest is request type for the Query/Params RPC method.
type QueryParamsRequest struct {
}

func (m *QueryParamsRequest) Reset()         { *m = QueryParamsRequest{} }
func (m *QueryParamsRequest) String() string { return proto.CompactTextString(m) }
func (*QueryParamsRequest) ProtoMessage()    {}
func (*QueryParamsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{0}
}
func (m *QueryParamsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryParamsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryParamsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryParamsRequest.Merge(m, src)
}
func (m *QueryParamsRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryParamsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryParamsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryParamsRequest proto.InternalMessageInfo

// QueryParamsResponse is response type for the Query/Params RPC method.
type QueryParamsResponse struct {
	// params holds all the parameters of this module.
	Params Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryParamsResponse) ProtoMessage()    {}
func (*QueryParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{1}
}
func (m *QueryParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryParamsResponse.Merge(m, src)
}
func (m *QueryParamsResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryParamsResponse proto.InternalMessageInfo

func (m *QueryParamsResponse) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

type QueryGetApplicationRequest struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (m *QueryGetApplicationRequest) Reset()         { *m = QueryGetApplicationRequest{} }
func (m *QueryGetApplicationRequest) String() string { return proto.CompactTextString(m) }
func (*QueryGetApplicationRequest) ProtoMessage()    {}
func (*QueryGetApplicationRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{2}
}
func (m *QueryGetApplicationRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryGetApplicationRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryGetApplicationRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryGetApplicationRequest.Merge(m, src)
}
func (m *QueryGetApplicationRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryGetApplicationRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryGetApplicationRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryGetApplicationRequest proto.InternalMessageInfo

func (m *QueryGetApplicationRequest) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type QueryGetApplicationResponse struct {
	Application Application `protobuf:"bytes,1,opt,name=application,proto3" json:"application"`
}

func (m *QueryGetApplicationResponse) Reset()         { *m = QueryGetApplicationResponse{} }
func (m *QueryGetApplicationResponse) String() string { return proto.CompactTextString(m) }
func (*QueryGetApplicationResponse) ProtoMessage()    {}
func (*QueryGetApplicationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{3}
}
func (m *QueryGetApplicationResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryGetApplicationResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryGetApplicationResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryGetApplicationResponse.Merge(m, src)
}
func (m *QueryGetApplicationResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryGetApplicationResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryGetApplicationResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryGetApplicationResponse proto.InternalMessageInfo

func (m *QueryGetApplicationResponse) GetApplication() Application {
	if m != nil {
		return m.Application
	}
	return Application{}
}

type QueryAllApplicationsRequest struct {
	Pagination *query.PageRequest `protobuf:"bytes,1,opt,name=pagination,proto3" json:"pagination,omitempty"`
	// TODO_MAINNET(@adshmh): rename this field to `gateway_address_delegated_to`
	// delegatee_gateway_address, if specified, filters the application list to only include those with delegation to the specified gateway address.
	DelegateeGatewayAddress string `protobuf:"bytes,2,opt,name=delegatee_gateway_address,json=delegateeGatewayAddress,proto3" json:"delegatee_gateway_address,omitempty"`
}

func (m *QueryAllApplicationsRequest) Reset()         { *m = QueryAllApplicationsRequest{} }
func (m *QueryAllApplicationsRequest) String() string { return proto.CompactTextString(m) }
func (*QueryAllApplicationsRequest) ProtoMessage()    {}
func (*QueryAllApplicationsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{4}
}
func (m *QueryAllApplicationsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryAllApplicationsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryAllApplicationsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryAllApplicationsRequest.Merge(m, src)
}
func (m *QueryAllApplicationsRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryAllApplicationsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryAllApplicationsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryAllApplicationsRequest proto.InternalMessageInfo

func (m *QueryAllApplicationsRequest) GetPagination() *query.PageRequest {
	if m != nil {
		return m.Pagination
	}
	return nil
}

func (m *QueryAllApplicationsRequest) GetDelegateeGatewayAddress() string {
	if m != nil {
		return m.DelegateeGatewayAddress
	}
	return ""
}

type QueryAllApplicationsResponse struct {
	Applications []Application       `protobuf:"bytes,1,rep,name=applications,proto3" json:"applications"`
	Pagination   *query.PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

func (m *QueryAllApplicationsResponse) Reset()         { *m = QueryAllApplicationsResponse{} }
func (m *QueryAllApplicationsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryAllApplicationsResponse) ProtoMessage()    {}
func (*QueryAllApplicationsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5cf77e4e046ed3a7, []int{5}
}
func (m *QueryAllApplicationsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryAllApplicationsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *QueryAllApplicationsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryAllApplicationsResponse.Merge(m, src)
}
func (m *QueryAllApplicationsResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryAllApplicationsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryAllApplicationsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryAllApplicationsResponse proto.InternalMessageInfo

func (m *QueryAllApplicationsResponse) GetApplications() []Application {
	if m != nil {
		return m.Applications
	}
	return nil
}

func (m *QueryAllApplicationsResponse) GetPagination() *query.PageResponse {
	if m != nil {
		return m.Pagination
	}
	return nil
}

func init() {
	proto.RegisterType((*QueryParamsRequest)(nil), "poktroll.application.QueryParamsRequest")
	proto.RegisterType((*QueryParamsResponse)(nil), "poktroll.application.QueryParamsResponse")
	proto.RegisterType((*QueryGetApplicationRequest)(nil), "poktroll.application.QueryGetApplicationRequest")
	proto.RegisterType((*QueryGetApplicationResponse)(nil), "poktroll.application.QueryGetApplicationResponse")
	proto.RegisterType((*QueryAllApplicationsRequest)(nil), "poktroll.application.QueryAllApplicationsRequest")
	proto.RegisterType((*QueryAllApplicationsResponse)(nil), "poktroll.application.QueryAllApplicationsResponse")
}

func init() { proto.RegisterFile("poktroll/application/query.proto", fileDescriptor_5cf77e4e046ed3a7) }

var fileDescriptor_5cf77e4e046ed3a7 = []byte{
	// 574 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x94, 0x41, 0x6f, 0xd3, 0x3e,
	0x18, 0xc6, 0xeb, 0xfe, 0xf7, 0x2f, 0x9a, 0x8b, 0x84, 0x30, 0x95, 0x18, 0xa1, 0x0a, 0x5b, 0x0e,
	0xb0, 0x0d, 0x61, 0xaf, 0x05, 0x4d, 0x68, 0x17, 0x68, 0x0f, 0x54, 0x88, 0xcb, 0xc8, 0x81, 0x03,
	0x97, 0xc9, 0x6d, 0xad, 0x2c, 0x5a, 0x1a, 0x67, 0x89, 0xcb, 0xa8, 0x10, 0x17, 0xc4, 0x07, 0x98,
	0xc4, 0x27, 0xe0, 0xc6, 0x71, 0x87, 0x7d, 0x88, 0x1d, 0x27, 0x71, 0xd9, 0x09, 0xa1, 0x16, 0x89,
	0xaf, 0x81, 0x62, 0x3b, 0xcc, 0x65, 0xa6, 0xac, 0x97, 0xc8, 0x8d, 0x9f, 0xc7, 0xef, 0xef, 0x79,
	0xfd, 0x36, 0x70, 0x39, 0xe1, 0x7b, 0x22, 0xe5, 0x51, 0x44, 0x68, 0x92, 0x44, 0x61, 0x8f, 0x8a,
	0x90, 0xc7, 0x64, 0x7f, 0xc8, 0xd2, 0x11, 0x4e, 0x52, 0x2e, 0x38, 0xaa, 0x15, 0x0a, 0x6c, 0x28,
	0x9c, 0xeb, 0x74, 0x10, 0xc6, 0x9c, 0xc8, 0xa7, 0x12, 0x3a, 0xb5, 0x80, 0x07, 0x5c, 0x2e, 0x49,
	0xbe, 0xd2, 0x6f, 0xeb, 0x01, 0xe7, 0x41, 0xc4, 0x08, 0x4d, 0x42, 0x42, 0xe3, 0x98, 0x0b, 0xe9,
	0xcf, 0xf4, 0xee, 0x7a, 0x8f, 0x67, 0x03, 0x9e, 0x91, 0x2e, 0xcd, 0x98, 0xaa, 0x4a, 0xde, 0x34,
	0xba, 0x4c, 0xd0, 0x06, 0x49, 0x68, 0x10, 0xc6, 0x52, 0xac, 0xb5, 0xae, 0xa9, 0x2d, 0x54, 0x3d,
	0x1e, 0x16, 0xfb, 0x2b, 0xd6, 0x28, 0x09, 0x4d, 0xe9, 0xa0, 0x28, 0x67, 0x4f, 0x2b, 0x46, 0x09,
	0xd3, 0x0a, 0xaf, 0x06, 0xd1, 0xcb, 0x1c, 0x63, 0x5b, 0xda, 0x7c, 0xb6, 0x3f, 0x64, 0x99, 0xf0,
	0x5e, 0xc1, 0x1b, 0x53, 0x6f, 0xb3, 0x84, 0xc7, 0x19, 0x43, 0x4f, 0x60, 0x45, 0x1d, 0xbf, 0x04,
	0x96, 0xc1, 0x6a, 0xb5, 0x59, 0xc7, 0xb6, 0x5e, 0x61, 0xe5, 0x6a, 0x2f, 0x9e, 0x7c, 0xbb, 0x53,
	0xfa, 0xf2, 0xf3, 0x68, 0x1d, 0xf8, 0xda, 0xe6, 0x6d, 0x42, 0x47, 0x9e, 0xdb, 0x61, 0xa2, 0x75,
	0x6e, 0xd0, 0x55, 0xd1, 0x12, 0xbc, 0x42, 0xfb, 0xfd, 0x94, 0x65, 0xea, 0xfc, 0x45, 0xbf, 0xf8,
	0xe9, 0xed, 0xc2, 0xdb, 0x56, 0x9f, 0xe6, 0x7a, 0x0e, 0xab, 0x46, 0x7d, 0x0d, 0xb7, 0x62, 0x87,
	0x33, 0xfc, 0xed, 0x85, 0x9c, 0xd0, 0x37, 0xbd, 0xde, 0x67, 0xa0, 0x4b, 0xb5, 0xa2, 0xc8, 0x90,
	0x16, 0x9d, 0x41, 0xcf, 0x20, 0x3c, 0xbf, 0x28, 0x5d, 0xe9, 0x2e, 0x56, 0x37, 0x85, 0xf3, 0x9b,
	0xc2, 0x6a, 0x96, 0xf4, 0x7d, 0xe1, 0x6d, 0x1a, 0x30, 0xed, 0xf5, 0x0d, 0x27, 0xda, 0x82, 0xb7,
	0xfa, 0x2c, 0x62, 0x01, 0x15, 0x8c, 0xed, 0xe4, 0xcf, 0x03, 0x3a, 0xda, 0x29, 0xd2, 0x97, 0x65,
	0xfa, 0x9b, 0xbf, 0x05, 0x1d, 0xb5, 0xdf, 0xd2, 0xdd, 0x38, 0x06, 0xb0, 0x6e, 0x67, 0xd4, 0xfd,
	0x78, 0x01, 0xaf, 0x1a, 0x99, 0xf2, 0x6e, 0xfe, 0x37, 0x4f, 0x43, 0xa6, 0xcc, 0xa8, 0x33, 0x95,
	0xb8, 0x2c, 0x13, 0xdf, 0xfb, 0x67, 0x62, 0x45, 0x62, 0x46, 0x6e, 0x7e, 0x5c, 0x80, 0xff, 0x4b,
	0x6c, 0x74, 0x08, 0x60, 0x45, 0x0d, 0x09, 0x5a, 0xb5, 0x43, 0x5d, 0x9c, 0x49, 0x67, 0xed, 0x12,
	0x4a, 0x55, 0xd5, 0x6b, 0x7c, 0xf8, 0xfa, 0xe3, 0x53, 0xf9, 0x3e, 0x5a, 0x23, 0xb9, 0xe5, 0x41,
	0xcc, 0xc4, 0x01, 0x4f, 0xf7, 0xc8, 0x8c, 0xff, 0x0b, 0x3a, 0x06, 0xb0, 0x6a, 0x74, 0x02, 0x6d,
	0xcc, 0xa8, 0x66, 0x9d, 0x5e, 0xa7, 0x31, 0x87, 0x43, 0x73, 0x3e, 0x95, 0x9c, 0x5b, 0xe8, 0xf1,
	0x25, 0x38, 0xcd, 0xf5, 0x3b, 0x3d, 0x28, 0xef, 0xd1, 0x11, 0x80, 0xd7, 0xfe, 0x98, 0x02, 0x34,
	0x0b, 0xc4, 0x3e, 0xd5, 0x4e, 0x73, 0x1e, 0x8b, 0x86, 0xdf, 0x94, 0xf0, 0x1b, 0x08, 0xcf, 0x07,
	0xdf, 0xf6, 0x4f, 0xc6, 0x2e, 0x38, 0x1d, 0xbb, 0xe0, 0x6c, 0xec, 0x82, 0xef, 0x63, 0x17, 0x1c,
	0x4e, 0xdc, 0xd2, 0xe9, 0xc4, 0x2d, 0x9d, 0x4d, 0xdc, 0xd2, 0xeb, 0x47, 0x41, 0x28, 0x76, 0x87,
	0x5d, 0xdc, 0xe3, 0x83, 0xbf, 0x9c, 0xfb, 0xf6, 0xe2, 0xb7, 0xac, 0x5b, 0x91, 0x1f, 0xb3, 0x87,
	0xbf, 0x02, 0x00, 0x00, 0xff, 0xff, 0x0c, 0x43, 0x1f, 0xbc, 0xde, 0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type QueryClient interface {
	// Parameters queries the parameters of the module.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// Queries a list of Application items.
	Application(ctx context.Context, in *QueryGetApplicationRequest, opts ...grpc.CallOption) (*QueryGetApplicationResponse, error)
	AllApplications(ctx context.Context, in *QueryAllApplicationsRequest, opts ...grpc.CallOption) (*QueryAllApplicationsResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/poktroll.application.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Application(ctx context.Context, in *QueryGetApplicationRequest, opts ...grpc.CallOption) (*QueryGetApplicationResponse, error) {
	out := new(QueryGetApplicationResponse)
	err := c.cc.Invoke(ctx, "/poktroll.application.Query/Application", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AllApplications(ctx context.Context, in *QueryAllApplicationsRequest, opts ...grpc.CallOption) (*QueryAllApplicationsResponse, error) {
	out := new(QueryAllApplicationsResponse)
	err := c.cc.Invoke(ctx, "/poktroll.application.Query/AllApplications", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
type QueryServer interface {
	// Parameters queries the parameters of the module.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	// Queries a list of Application items.
	Application(context.Context, *QueryGetApplicationRequest) (*QueryGetApplicationResponse, error)
	AllApplications(context.Context, *QueryAllApplicationsRequest) (*QueryAllApplicationsResponse, error)
}

// UnimplementedQueryServer can be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (*UnimplementedQueryServer) Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (*UnimplementedQueryServer) Application(ctx context.Context, req *QueryGetApplicationRequest) (*QueryGetApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Application not implemented")
}
func (*UnimplementedQueryServer) AllApplications(ctx context.Context, req *QueryAllApplicationsRequest) (*QueryAllApplicationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllApplications not implemented")
}

func RegisterQueryServer(s grpc1.Server, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/poktroll.application.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Application_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetApplicationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Application(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/poktroll.application.Query/Application",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Application(ctx, req.(*QueryGetApplicationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AllApplications_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllApplicationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllApplications(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/poktroll.application.Query/AllApplications",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllApplications(ctx, req.(*QueryAllApplicationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var Query_serviceDesc = _Query_serviceDesc
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "poktroll.application.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "Application",
			Handler:    _Query_Application_Handler,
		},
		{
			MethodName: "AllApplications",
			Handler:    _Query_AllApplications_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "poktroll/application/query.proto",
}

func (m *QueryParamsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryParamsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryParamsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *QueryParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QueryGetApplicationRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryGetApplicationRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryGetApplicationRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintQuery(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *QueryGetApplicationResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryGetApplicationResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryGetApplicationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Application.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QueryAllApplicationsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryAllApplicationsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryAllApplicationsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.DelegateeGatewayAddress) > 0 {
		i -= len(m.DelegateeGatewayAddress)
		copy(dAtA[i:], m.DelegateeGatewayAddress)
		i = encodeVarintQuery(dAtA, i, uint64(len(m.DelegateeGatewayAddress)))
		i--
		dAtA[i] = 0x12
	}
	if m.Pagination != nil {
		{
			size, err := m.Pagination.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintQuery(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *QueryAllApplicationsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryAllApplicationsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryAllApplicationsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Pagination != nil {
		{
			size, err := m.Pagination.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintQuery(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Applications) > 0 {
		for iNdEx := len(m.Applications) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Applications[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintQuery(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintQuery(dAtA []byte, offset int, v uint64) int {
	offset -= sovQuery(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *QueryParamsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *QueryParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func (m *QueryGetApplicationRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovQuery(uint64(l))
	}
	return n
}

func (m *QueryGetApplicationResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Application.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func (m *QueryAllApplicationsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Pagination != nil {
		l = m.Pagination.Size()
		n += 1 + l + sovQuery(uint64(l))
	}
	l = len(m.DelegateeGatewayAddress)
	if l > 0 {
		n += 1 + l + sovQuery(uint64(l))
	}
	return n
}

func (m *QueryAllApplicationsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Applications) > 0 {
		for _, e := range m.Applications {
			l = e.Size()
			n += 1 + l + sovQuery(uint64(l))
		}
	}
	if m.Pagination != nil {
		l = m.Pagination.Size()
		n += 1 + l + sovQuery(uint64(l))
	}
	return n
}

func sovQuery(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQuery(x uint64) (n int) {
	return sovQuery(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *QueryParamsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryParamsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryParamsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryParamsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryGetApplicationRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryGetApplicationRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryGetApplicationRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryGetApplicationResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryGetApplicationResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryGetApplicationResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Application", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Application.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryAllApplicationsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryAllApplicationsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryAllApplicationsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pagination", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Pagination == nil {
				m.Pagination = &query.PageRequest{}
			}
			if err := m.Pagination.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DelegateeGatewayAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DelegateeGatewayAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryAllApplicationsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryAllApplicationsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryAllApplicationsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Applications", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Applications = append(m.Applications, Application{})
			if err := m.Applications[len(m.Applications)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pagination", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Pagination == nil {
				m.Pagination = &query.PageResponse{}
			}
			if err := m.Pagination.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipQuery(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthQuery
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQuery
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQuery
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQuery        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQuery          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQuery = fmt.Errorf("proto: unexpected end of group")
)
