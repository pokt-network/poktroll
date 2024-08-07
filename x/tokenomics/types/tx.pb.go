// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: poktroll/tokenomics/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
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

// MsgUpdateParams is the Msg/UpdateParams request type to update all params at once.
type MsgUpdateParams struct {
	// authority is the address that controls the module (defaults to x/gov unless overwritten).
	Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	// params defines the x/tokenomics parameters to update.
	// NOTE: All parameters must be supplied.
	Params Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParams) ProtoMessage()    {}
func (*MsgUpdateParams) Descriptor() ([]byte, []int) {
	return fileDescriptor_aa0f2fdbd9b6d7eb, []int{0}
}
func (m *MsgUpdateParams) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParams.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParams) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParams.Merge(m, src)
}
func (m *MsgUpdateParams) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParams) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParams.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParams proto.InternalMessageInfo

func (m *MsgUpdateParams) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

func (m *MsgUpdateParams) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

// MsgUpdateParamsResponse defines the response structure for executing a
// MsgUpdateParams message.
type MsgUpdateParamsResponse struct {
}

func (m *MsgUpdateParamsResponse) Reset()         { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParamsResponse) ProtoMessage()    {}
func (*MsgUpdateParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_aa0f2fdbd9b6d7eb, []int{1}
}
func (m *MsgUpdateParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParamsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParamsResponse.Merge(m, src)
}
func (m *MsgUpdateParamsResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParamsResponse proto.InternalMessageInfo

// MsgUpdateParam is the Msg/UpdateParam request type to update a single param.
type MsgUpdateParam struct {
	// authority is the address that controls the module (defaults to x/gov unless overwritten).
	Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	// The (name, as_type) tuple must match the corresponding name and type as
	// specified in the `Params`` message in `proof/params.proto.`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are valid to be assigned to AsType:
	//	*MsgUpdateParam_AsString
	//	*MsgUpdateParam_AsInt64
	//	*MsgUpdateParam_AsBytes
	AsType isMsgUpdateParam_AsType `protobuf_oneof:"as_type"`
}

func (m *MsgUpdateParam) Reset()         { *m = MsgUpdateParam{} }
func (m *MsgUpdateParam) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParam) ProtoMessage()    {}
func (*MsgUpdateParam) Descriptor() ([]byte, []int) {
	return fileDescriptor_aa0f2fdbd9b6d7eb, []int{2}
}
func (m *MsgUpdateParam) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParam) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParam.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParam) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParam.Merge(m, src)
}
func (m *MsgUpdateParam) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParam) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParam.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParam proto.InternalMessageInfo

type isMsgUpdateParam_AsType interface {
	isMsgUpdateParam_AsType()
	MarshalTo([]byte) (int, error)
	Size() int
}

type MsgUpdateParam_AsString struct {
	AsString string `protobuf:"bytes,3,opt,name=as_string,json=asString,proto3,oneof" json:"as_string"`
}
type MsgUpdateParam_AsInt64 struct {
	AsInt64 int64 `protobuf:"varint,6,opt,name=as_int64,json=asInt64,proto3,oneof" json:"as_int64"`
}
type MsgUpdateParam_AsBytes struct {
	AsBytes []byte `protobuf:"bytes,7,opt,name=as_bytes,json=asBytes,proto3,oneof" json:"as_bytes"`
}

func (*MsgUpdateParam_AsString) isMsgUpdateParam_AsType() {}
func (*MsgUpdateParam_AsInt64) isMsgUpdateParam_AsType()  {}
func (*MsgUpdateParam_AsBytes) isMsgUpdateParam_AsType()  {}

func (m *MsgUpdateParam) GetAsType() isMsgUpdateParam_AsType {
	if m != nil {
		return m.AsType
	}
	return nil
}

func (m *MsgUpdateParam) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

func (m *MsgUpdateParam) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *MsgUpdateParam) GetAsString() string {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsString); ok {
		return x.AsString
	}
	return ""
}

func (m *MsgUpdateParam) GetAsInt64() int64 {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsInt64); ok {
		return x.AsInt64
	}
	return 0
}

func (m *MsgUpdateParam) GetAsBytes() []byte {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsBytes); ok {
		return x.AsBytes
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*MsgUpdateParam) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*MsgUpdateParam_AsString)(nil),
		(*MsgUpdateParam_AsInt64)(nil),
		(*MsgUpdateParam_AsBytes)(nil),
	}
}

// MsgUpdateParamResponse defines the response structure for executing a
// MsgUpdateParam message after a single param update.
type MsgUpdateParamResponse struct {
	Params *Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params,omitempty"`
}

func (m *MsgUpdateParamResponse) Reset()         { *m = MsgUpdateParamResponse{} }
func (m *MsgUpdateParamResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParamResponse) ProtoMessage()    {}
func (*MsgUpdateParamResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_aa0f2fdbd9b6d7eb, []int{3}
}
func (m *MsgUpdateParamResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParamResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParamResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParamResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParamResponse.Merge(m, src)
}
func (m *MsgUpdateParamResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParamResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParamResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParamResponse proto.InternalMessageInfo

func (m *MsgUpdateParamResponse) GetParams() *Params {
	if m != nil {
		return m.Params
	}
	return nil
}

func init() {
	proto.RegisterType((*MsgUpdateParams)(nil), "poktroll.tokenomics.MsgUpdateParams")
	proto.RegisterType((*MsgUpdateParamsResponse)(nil), "poktroll.tokenomics.MsgUpdateParamsResponse")
	proto.RegisterType((*MsgUpdateParam)(nil), "poktroll.tokenomics.MsgUpdateParam")
	proto.RegisterType((*MsgUpdateParamResponse)(nil), "poktroll.tokenomics.MsgUpdateParamResponse")
}

func init() { proto.RegisterFile("poktroll/tokenomics/tx.proto", fileDescriptor_aa0f2fdbd9b6d7eb) }

var fileDescriptor_aa0f2fdbd9b6d7eb = []byte{
	// 505 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x53, 0xbf, 0x6f, 0xd3, 0x40,
	0x14, 0xf6, 0x11, 0x48, 0xea, 0x6b, 0x28, 0xc2, 0x54, 0xd4, 0x35, 0xc8, 0xb1, 0x02, 0x48, 0x21,
	0xb4, 0xb6, 0x68, 0x50, 0x85, 0x3a, 0x20, 0xe1, 0xa9, 0x0c, 0x91, 0x90, 0x11, 0x0b, 0x8b, 0x75,
	0x49, 0x4e, 0xae, 0x95, 0xda, 0x67, 0xf9, 0x5d, 0xa1, 0xd9, 0x10, 0x23, 0x13, 0x7f, 0x06, 0x63,
	0x06, 0xfe, 0x03, 0x96, 0x8e, 0x85, 0x89, 0x29, 0x42, 0xc9, 0x10, 0xa9, 0xff, 0x02, 0x0b, 0xf2,
	0xd9, 0x71, 0x7e, 0xc8, 0x12, 0x11, 0x4b, 0xf2, 0xee, 0xfb, 0xbe, 0xf7, 0xee, 0xbe, 0xf7, 0x9e,
	0xf1, 0xfd, 0x88, 0xf5, 0x79, 0xcc, 0x4e, 0x4f, 0x2d, 0xce, 0xfa, 0x34, 0x64, 0x81, 0xdf, 0x05,
	0x8b, 0x9f, 0x9b, 0x51, 0xcc, 0x38, 0x53, 0xee, 0xcc, 0x58, 0x73, 0xce, 0x6a, 0xb7, 0x49, 0xe0,
	0x87, 0xcc, 0x12, 0xbf, 0xa9, 0x4e, 0xdb, 0xe9, 0x32, 0x08, 0x18, 0x58, 0x01, 0x78, 0xd6, 0xfb,
	0xa7, 0xc9, 0x5f, 0x46, 0xec, 0xa6, 0x84, 0x2b, 0x4e, 0x56, 0x7a, 0xc8, 0xa8, 0x6d, 0x8f, 0x79,
	0x2c, 0xc5, 0x93, 0x28, 0x43, 0x8d, 0xa2, 0xf7, 0x44, 0x24, 0x26, 0x41, 0x96, 0x57, 0xff, 0x8e,
	0xf0, 0xad, 0x36, 0x78, 0x6f, 0xa3, 0x1e, 0xe1, 0xf4, 0xb5, 0x60, 0x94, 0x43, 0x2c, 0x93, 0x33,
	0x7e, 0xc2, 0x62, 0x9f, 0x0f, 0x54, 0x64, 0xa0, 0x86, 0x6c, 0xab, 0x3f, 0xbf, 0xed, 0x6f, 0x67,
	0x17, 0xbe, 0xec, 0xf5, 0x62, 0x0a, 0xf0, 0x86, 0xc7, 0x7e, 0xe8, 0x39, 0x73, 0xa9, 0xf2, 0x02,
	0x97, 0xd3, 0xda, 0xea, 0x35, 0x03, 0x35, 0x36, 0x0f, 0xee, 0x99, 0x05, 0x86, 0xcd, 0xf4, 0x12,
	0x5b, 0xbe, 0x18, 0xd5, 0xa4, 0xaf, 0xd3, 0x61, 0x13, 0x39, 0x59, 0xd6, 0xd1, 0xf3, 0x4f, 0xd3,
	0x61, 0x73, 0x5e, 0xef, 0xf3, 0x74, 0xd8, 0x7c, 0x94, 0x1b, 0x38, 0x5f, 0xb4, 0xb0, 0xf2, 0xe2,
	0xfa, 0x2e, 0xde, 0x59, 0x81, 0x1c, 0x0a, 0x11, 0x0b, 0x81, 0xd6, 0xff, 0x20, 0xbc, 0xb5, 0xcc,
	0xfd, 0xb7, 0x3f, 0x05, 0x5f, 0x0f, 0x49, 0x40, 0x85, 0x3b, 0xd9, 0x11, 0xb1, 0xb2, 0x87, 0x65,
	0x02, 0x2e, 0x08, 0xad, 0x5a, 0x12, 0xb5, 0x6e, 0x5e, 0x8d, 0x6a, 0x73, 0xf0, 0x58, 0x72, 0x36,
	0x48, 0x56, 0x4c, 0x79, 0x8c, 0x37, 0x08, 0xb8, 0x7e, 0xc8, 0x0f, 0x9f, 0xa9, 0x65, 0x03, 0x35,
	0x4a, 0x76, 0xf5, 0x6a, 0x54, 0xcb, 0xb1, 0x63, 0xc9, 0xa9, 0x10, 0x78, 0x95, 0x84, 0x99, 0xb4,
	0x33, 0xe0, 0x14, 0xd4, 0x8a, 0x81, 0x1a, 0xd5, 0x5c, 0x2a, 0xb0, 0x54, 0x6a, 0x27, 0xe1, 0xd1,
	0xd6, 0x72, 0xdf, 0x6c, 0x19, 0x57, 0x08, 0xb8, 0x7c, 0x10, 0xd1, 0x7a, 0x1b, 0xdf, 0x5d, 0x36,
	0x3f, 0xeb, 0x8b, 0xd2, 0xca, 0x87, 0x85, 0xfe, 0x39, 0xac, 0xd9, 0x84, 0x0e, 0x7e, 0x20, 0x5c,
	0x6a, 0x83, 0xa7, 0x74, 0x70, 0x75, 0x69, 0x63, 0x1e, 0x16, 0x26, 0xaf, 0x8c, 0x44, 0xdb, 0x5b,
	0x47, 0x95, 0x3f, 0xd0, 0xc5, 0x9b, 0x8b, 0x43, 0x7b, 0xb0, 0x46, 0xb2, 0xf6, 0x64, 0x0d, 0xd1,
	0xec, 0x02, 0xed, 0xc6, 0xc7, 0x64, 0xfb, 0xec, 0xf6, 0xc5, 0x58, 0x47, 0x97, 0x63, 0x1d, 0xfd,
	0x1e, 0xeb, 0xe8, 0xcb, 0x44, 0x97, 0x2e, 0x27, 0xba, 0xf4, 0x6b, 0xa2, 0x4b, 0xef, 0x5a, 0x9e,
	0xcf, 0x4f, 0xce, 0x3a, 0x66, 0x97, 0x05, 0x56, 0x52, 0x77, 0x3f, 0xa4, 0xfc, 0x03, 0x8b, 0xfb,
	0x56, 0xf1, 0x52, 0x26, 0x0d, 0x87, 0x4e, 0x59, 0x7c, 0x57, 0xad, 0xbf, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x00, 0xfb, 0x67, 0xbf, 0x0b, 0x04, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	UpdateParam(ctx context.Context, in *MsgUpdateParam, opts ...grpc.CallOption) (*MsgUpdateParamResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, "/poktroll.tokenomics.Msg/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateParam(ctx context.Context, in *MsgUpdateParam, opts ...grpc.CallOption) (*MsgUpdateParamResponse, error) {
	out := new(MsgUpdateParamResponse)
	err := c.cc.Invoke(ctx, "/poktroll.tokenomics.Msg/UpdateParam", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	UpdateParam(context.Context, *MsgUpdateParam) (*MsgUpdateParamResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) UpdateParams(ctx context.Context, req *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (*UnimplementedMsgServer) UpdateParam(ctx context.Context, req *MsgUpdateParam) (*MsgUpdateParamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParam not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
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
		FullMethod: "/poktroll.tokenomics.Msg/UpdateParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
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
		FullMethod: "/poktroll.tokenomics.Msg/UpdateParam",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParam(ctx, req.(*MsgUpdateParam))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "poktroll.tokenomics.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "UpdateParam",
			Handler:    _Msg_UpdateParam_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "poktroll/tokenomics/tx.proto",
}

func (m *MsgUpdateParams) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParams) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParams) MarshalToSizedBuffer(dAtA []byte) (int, error) {
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
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgUpdateParam) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParam) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.AsType != nil {
		{
			size := m.AsType.Size()
			i -= size
			if _, err := m.AsType.MarshalTo(dAtA[i:]); err != nil {
				return 0, err
			}
		}
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateParam_AsString) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsString) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= len(m.AsString)
	copy(dAtA[i:], m.AsString)
	i = encodeVarintTx(dAtA, i, uint64(len(m.AsString)))
	i--
	dAtA[i] = 0x1a
	return len(dAtA) - i, nil
}
func (m *MsgUpdateParam_AsInt64) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsInt64) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i = encodeVarintTx(dAtA, i, uint64(m.AsInt64))
	i--
	dAtA[i] = 0x30
	return len(dAtA) - i, nil
}
func (m *MsgUpdateParam_AsBytes) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsBytes) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.AsBytes != nil {
		i -= len(m.AsBytes)
		copy(dAtA[i:], m.AsBytes)
		i = encodeVarintTx(dAtA, i, uint64(len(m.AsBytes)))
		i--
		dAtA[i] = 0x3a
	}
	return len(dAtA) - i, nil
}
func (m *MsgUpdateParamResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParamResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParamResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Params != nil {
		{
			size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgUpdateParams) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Authority)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Params.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgUpdateParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgUpdateParam) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Authority)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.AsType != nil {
		n += m.AsType.Size()
	}
	return n
}

func (m *MsgUpdateParam_AsString) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.AsString)
	n += 1 + l + sovTx(uint64(l))
	return n
}
func (m *MsgUpdateParam_AsInt64) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 1 + sovTx(uint64(m.AsInt64))
	return n
}
func (m *MsgUpdateParam_AsBytes) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.AsBytes != nil {
		l = len(m.AsBytes)
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}
func (m *MsgUpdateParamResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Params != nil {
		l = m.Params.Size()
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgUpdateParams) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateParams: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParams: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Authority", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
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
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgUpdateParamsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgUpdateParam) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateParam: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParam: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Authority", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AsString", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AsType = &MsgUpdateParam_AsString{string(dAtA[iNdEx:postIndex])}
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AsInt64", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.AsType = &MsgUpdateParam_AsInt64{v}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AsBytes", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			v := make([]byte, postIndex-iNdEx)
			copy(v, dAtA[iNdEx:postIndex])
			m.AsType = &MsgUpdateParam_AsBytes{v}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgUpdateParamResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgUpdateParamResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParamResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Params == nil {
				m.Params = &Params{}
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
