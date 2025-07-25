// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pocket/tokenomics/tx.proto

package types

import (
	context "context"
	encoding_binary "encoding/binary"
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
	// params defines the tokenomics parameters to update.
	Params Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParams) ProtoMessage()    {}
func (*MsgUpdateParams) Descriptor() ([]byte, []int) {
	return fileDescriptor_df88dc3fd9e72965, []int{0}
}
func (m *MsgUpdateParams) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
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

// MsgUpdateParamsResponse defines the response structure for executing a MsgUpdateParams message.
type MsgUpdateParamsResponse struct {
}

func (m *MsgUpdateParamsResponse) Reset()         { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParamsResponse) ProtoMessage()    {}
func (*MsgUpdateParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_df88dc3fd9e72965, []int{1}
}
func (m *MsgUpdateParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
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
	// The (name, as_type) tuple must match the corresponding name and type as specified in the `Params` message in `proof/params.proto.`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are valid to be assigned to AsType:
	//	*MsgUpdateParam_AsMintAllocationPercentages
	//	*MsgUpdateParam_AsString
	//	*MsgUpdateParam_AsFloat
	//	*MsgUpdateParam_AsMintEqualsBurnClaimDistribution
	AsType isMsgUpdateParam_AsType `protobuf_oneof:"as_type"`
}

func (m *MsgUpdateParam) Reset()         { *m = MsgUpdateParam{} }
func (m *MsgUpdateParam) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParam) ProtoMessage()    {}
func (*MsgUpdateParam) Descriptor() ([]byte, []int) {
	return fileDescriptor_df88dc3fd9e72965, []int{2}
}
func (m *MsgUpdateParam) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParam) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
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

type MsgUpdateParam_AsMintAllocationPercentages struct {
	AsMintAllocationPercentages *MintAllocationPercentages `protobuf:"bytes,3,opt,name=as_mint_allocation_percentages,json=asMintAllocationPercentages,proto3,oneof" json:"as_mint_allocation_percentages" yaml:"as_mint_allocation_percentages"`
}
type MsgUpdateParam_AsString struct {
	AsString string `protobuf:"bytes,4,opt,name=as_string,json=asString,proto3,oneof" json:"as_string"`
}
type MsgUpdateParam_AsFloat struct {
	AsFloat float64 `protobuf:"fixed64,5,opt,name=as_float,json=asFloat,proto3,oneof" json:"as_float"`
}
type MsgUpdateParam_AsMintEqualsBurnClaimDistribution struct {
	AsMintEqualsBurnClaimDistribution *MintEqualsBurnClaimDistribution `protobuf:"bytes,6,opt,name=as_mint_equals_burn_claim_distribution,json=asMintEqualsBurnClaimDistribution,proto3,oneof" json:"as_mint_equals_burn_claim_distribution" yaml:"as_mint_equals_burn_claim_distribution"`
}

func (*MsgUpdateParam_AsMintAllocationPercentages) isMsgUpdateParam_AsType()       {}
func (*MsgUpdateParam_AsString) isMsgUpdateParam_AsType()                          {}
func (*MsgUpdateParam_AsFloat) isMsgUpdateParam_AsType()                           {}
func (*MsgUpdateParam_AsMintEqualsBurnClaimDistribution) isMsgUpdateParam_AsType() {}

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

func (m *MsgUpdateParam) GetAsMintAllocationPercentages() *MintAllocationPercentages {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsMintAllocationPercentages); ok {
		return x.AsMintAllocationPercentages
	}
	return nil
}

func (m *MsgUpdateParam) GetAsString() string {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsString); ok {
		return x.AsString
	}
	return ""
}

func (m *MsgUpdateParam) GetAsFloat() float64 {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsFloat); ok {
		return x.AsFloat
	}
	return 0
}

func (m *MsgUpdateParam) GetAsMintEqualsBurnClaimDistribution() *MintEqualsBurnClaimDistribution {
	if x, ok := m.GetAsType().(*MsgUpdateParam_AsMintEqualsBurnClaimDistribution); ok {
		return x.AsMintEqualsBurnClaimDistribution
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*MsgUpdateParam) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*MsgUpdateParam_AsMintAllocationPercentages)(nil),
		(*MsgUpdateParam_AsString)(nil),
		(*MsgUpdateParam_AsFloat)(nil),
		(*MsgUpdateParam_AsMintEqualsBurnClaimDistribution)(nil),
	}
}

// MsgUpdateParamResponse defines the response structure for executing a MsgUpdateParam message after a single param update.
type MsgUpdateParamResponse struct {
}

func (m *MsgUpdateParamResponse) Reset()         { *m = MsgUpdateParamResponse{} }
func (m *MsgUpdateParamResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParamResponse) ProtoMessage()    {}
func (*MsgUpdateParamResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_df88dc3fd9e72965, []int{3}
}
func (m *MsgUpdateParamResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParamResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
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

func init() {
	proto.RegisterType((*MsgUpdateParams)(nil), "pocket.tokenomics.MsgUpdateParams")
	proto.RegisterType((*MsgUpdateParamsResponse)(nil), "pocket.tokenomics.MsgUpdateParamsResponse")
	proto.RegisterType((*MsgUpdateParam)(nil), "pocket.tokenomics.MsgUpdateParam")
	proto.RegisterType((*MsgUpdateParamResponse)(nil), "pocket.tokenomics.MsgUpdateParamResponse")
}

func init() { proto.RegisterFile("pocket/tokenomics/tx.proto", fileDescriptor_df88dc3fd9e72965) }

var fileDescriptor_df88dc3fd9e72965 = []byte{
	// 631 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0xcd, 0x4f, 0xd4, 0x4e,
	0x18, 0xee, 0xc0, 0xf2, 0xb1, 0x03, 0x3f, 0x7e, 0xd2, 0x10, 0x29, 0x35, 0x69, 0xa1, 0x46, 0x03,
	0x1b, 0xd8, 0x46, 0x48, 0x38, 0x10, 0x2f, 0xd4, 0x8f, 0x10, 0x0d, 0x09, 0xd6, 0x78, 0x91, 0xc4,
	0x66, 0xb6, 0x3b, 0x96, 0x66, 0xdb, 0x99, 0xda, 0x99, 0x2a, 0xdc, 0x8c, 0x47, 0x4f, 0xfe, 0x19,
	0x1e, 0x39, 0x78, 0xd7, 0x23, 0xde, 0x88, 0x27, 0x4e, 0x8d, 0x59, 0x4c, 0x48, 0xf6, 0xc8, 0xd1,
	0x93, 0xe9, 0xb4, 0xb0, 0x94, 0xaf, 0x35, 0x5e, 0x76, 0x67, 0xde, 0xf7, 0x79, 0xde, 0xf7, 0x99,
	0x27, 0xef, 0x5b, 0xa8, 0x46, 0xd4, 0x6d, 0x61, 0x6e, 0x72, 0xda, 0xc2, 0x84, 0x86, 0xbe, 0xcb,
	0x4c, 0xbe, 0x5d, 0x8f, 0x62, 0xca, 0xa9, 0x3c, 0x9e, 0xe7, 0xea, 0xdd, 0x9c, 0x3a, 0x8e, 0x42,
	0x9f, 0x50, 0x53, 0xfc, 0xe6, 0x28, 0x75, 0xd2, 0xa5, 0x2c, 0xa4, 0xcc, 0x0c, 0x99, 0x67, 0xbe,
	0xbd, 0x97, 0xfd, 0x15, 0x89, 0xa9, 0x3c, 0xe1, 0x88, 0x9b, 0x99, 0x5f, 0x8a, 0xd4, 0x84, 0x47,
	0x3d, 0x9a, 0xc7, 0xb3, 0x53, 0x11, 0xd5, 0x2e, 0x6a, 0x89, 0x50, 0x8c, 0xc2, 0x82, 0x65, 0x7c,
	0x05, 0xf0, 0xff, 0x75, 0xe6, 0xbd, 0x88, 0x9a, 0x88, 0xe3, 0x0d, 0x91, 0x91, 0x97, 0x61, 0x15,
	0x25, 0x7c, 0x8b, 0xc6, 0x3e, 0xdf, 0x51, 0xc0, 0x34, 0x98, 0xad, 0x5a, 0xca, 0x8f, 0x2f, 0x0b,
	0x13, 0x45, 0xbb, 0xd5, 0x66, 0x33, 0xc6, 0x8c, 0x3d, 0xe7, 0xb1, 0x4f, 0x3c, 0xbb, 0x0b, 0x95,
	0xef, 0xc3, 0xc1, 0xbc, 0xb6, 0xd2, 0x37, 0x0d, 0x66, 0x47, 0x16, 0xa7, 0xea, 0x17, 0x1e, 0x5b,
	0xcf, 0x5b, 0x58, 0xd5, 0xbd, 0x54, 0x97, 0x3e, 0x1f, 0xed, 0xd6, 0x80, 0x5d, 0x70, 0x56, 0x96,
	0x3f, 0x1c, 0xed, 0xd6, 0xba, 0xd5, 0x3e, 0x1e, 0xed, 0xd6, 0x6e, 0x17, 0xe2, 0xb7, 0xcf, 0xca,
	0x3f, 0xa7, 0xd6, 0xd0, 0xe1, 0xe4, 0xb9, 0x90, 0x8d, 0x59, 0x44, 0x09, 0xc3, 0x4f, 0x2a, 0xc3,
	0xe0, 0x46, 0x9f, 0xf1, 0xbb, 0x02, 0xc7, 0xca, 0x88, 0x7f, 0x7e, 0xa1, 0x0c, 0x2b, 0x04, 0x85,
	0x58, 0xbc, 0xaf, 0x6a, 0x8b, 0xb3, 0xfc, 0x0d, 0x40, 0x0d, 0x31, 0x27, 0xf4, 0x09, 0x77, 0x50,
	0x10, 0x50, 0x17, 0x71, 0x9f, 0x12, 0x27, 0xc2, 0xb1, 0x8b, 0x09, 0x47, 0x1e, 0x66, 0x4a, 0xbf,
	0xb0, 0x63, 0xfe, 0x12, 0x3b, 0xd6, 0x7d, 0xc2, 0x57, 0x4f, 0x49, 0x1b, 0x5d, 0x8e, 0xf5, 0xb4,
	0x93, 0xea, 0x3d, 0xea, 0x1e, 0xa7, 0xfa, 0x9d, 0x1d, 0x14, 0x06, 0x2b, 0xc6, 0xf5, 0x38, 0x63,
	0x4d, 0xb2, 0x6f, 0x21, 0x76, 0x65, 0x2f, 0x79, 0x1e, 0x56, 0x11, 0x73, 0x98, 0x78, 0xae, 0x52,
	0x11, 0x76, 0xfc, 0xd7, 0x49, 0xf5, 0x6e, 0x70, 0x4d, 0xb2, 0x87, 0x51, 0xe1, 0x87, 0x3c, 0x07,
	0x87, 0x11, 0x73, 0x5e, 0x07, 0x14, 0x71, 0x65, 0x60, 0x1a, 0xcc, 0x02, 0x6b, 0xb4, 0x93, 0xea,
	0xa7, 0xb1, 0x35, 0xc9, 0x1e, 0x42, 0xec, 0x71, 0x76, 0x94, 0x7f, 0x01, 0x78, 0xf7, 0x44, 0x1b,
	0x7e, 0x93, 0xa0, 0x80, 0x39, 0x8d, 0x24, 0x26, 0x8e, 0x1b, 0x20, 0x3f, 0x74, 0x9a, 0x7e, 0x56,
	0xbd, 0x91, 0x64, 0x72, 0x94, 0x41, 0xe1, 0xd1, 0xe2, 0x15, 0x1e, 0x3d, 0x12, 0x64, 0x2b, 0x89,
	0xc9, 0x83, 0x8c, 0xfa, 0xf0, 0x0c, 0xd3, 0xda, 0xec, 0xa4, 0xfa, 0x5f, 0x76, 0x39, 0x4e, 0xf5,
	0x85, 0xb2, 0x63, 0xd7, 0xe3, 0x33, 0xe7, 0x66, 0x72, 0xe7, 0xae, 0x51, 0xb0, 0x32, 0x56, 0x1e,
	0x5d, 0xab, 0x0a, 0x87, 0x10, 0x73, 0xf8, 0x4e, 0x84, 0x0d, 0x0d, 0xde, 0x2c, 0xcf, 0x5e, 0x79,
	0x38, 0x17, 0xbf, 0x03, 0xd8, 0xbf, 0xce, 0x3c, 0xf9, 0x15, 0x1c, 0x2d, 0xed, 0xa0, 0x71, 0x99,
	0x11, 0xe5, 0x31, 0x57, 0x6b, 0xbd, 0x31, 0x27, 0xdd, 0xe4, 0x4d, 0x38, 0x72, 0x76, 0x01, 0x66,
	0x7a, 0x52, 0xd5, 0xb9, 0x9e, 0x90, 0x93, 0xe2, 0xea, 0xc0, 0xfb, 0x6c, 0x93, 0xad, 0x67, 0x7b,
	0x6d, 0x0d, 0xec, 0xb7, 0x35, 0x70, 0xd0, 0xd6, 0xc0, 0xcf, 0xb6, 0x06, 0x3e, 0x1d, 0x6a, 0xd2,
	0xfe, 0xa1, 0x26, 0x1d, 0x1c, 0x6a, 0xd2, 0xcb, 0x25, 0xcf, 0xe7, 0x5b, 0x49, 0xa3, 0xee, 0xd2,
	0xd0, 0x8c, 0x68, 0x8b, 0x2f, 0x10, 0xcc, 0xdf, 0xd1, 0xb8, 0x25, 0x2e, 0x31, 0x0d, 0x82, 0xf2,
	0x9a, 0x67, 0xee, 0xb1, 0xc6, 0xa0, 0xf8, 0x4a, 0x2d, 0xfd, 0x09, 0x00, 0x00, 0xff, 0xff, 0x7c,
	0xcd, 0x0d, 0x7b, 0x53, 0x05, 0x00, 0x00,
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
	err := c.cc.Invoke(ctx, "/pocket.tokenomics.Msg/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateParam(ctx context.Context, in *MsgUpdateParam, opts ...grpc.CallOption) (*MsgUpdateParamResponse, error) {
	out := new(MsgUpdateParamResponse)
	err := c.cc.Invoke(ctx, "/pocket.tokenomics.Msg/UpdateParam", in, out, opts...)
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
		FullMethod: "/pocket.tokenomics.Msg/UpdateParams",
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
		FullMethod: "/pocket.tokenomics.Msg/UpdateParam",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParam(ctx, req.(*MsgUpdateParam))
	}
	return interceptor(ctx, in, info, handler)
}

var Msg_serviceDesc = _Msg_serviceDesc
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pocket.tokenomics.Msg",
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
	Metadata: "pocket/tokenomics/tx.proto",
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

func (m *MsgUpdateParam_AsMintAllocationPercentages) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsMintAllocationPercentages) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.AsMintAllocationPercentages != nil {
		{
			size, err := m.AsMintAllocationPercentages.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
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
	dAtA[i] = 0x22
	return len(dAtA) - i, nil
}
func (m *MsgUpdateParam_AsFloat) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsFloat) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= 8
	encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(math.Float64bits(float64(m.AsFloat))))
	i--
	dAtA[i] = 0x29
	return len(dAtA) - i, nil
}
func (m *MsgUpdateParam_AsMintEqualsBurnClaimDistribution) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParam_AsMintEqualsBurnClaimDistribution) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	if m.AsMintEqualsBurnClaimDistribution != nil {
		{
			size, err := m.AsMintEqualsBurnClaimDistribution.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTx(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x32
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

func (m *MsgUpdateParam_AsMintAllocationPercentages) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.AsMintAllocationPercentages != nil {
		l = m.AsMintAllocationPercentages.Size()
		n += 1 + l + sovTx(uint64(l))
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
func (m *MsgUpdateParam_AsFloat) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 9
	return n
}
func (m *MsgUpdateParam_AsMintEqualsBurnClaimDistribution) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.AsMintEqualsBurnClaimDistribution != nil {
		l = m.AsMintEqualsBurnClaimDistribution.Size()
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
				return fmt.Errorf("proto: wrong wireType = %d for field AsMintAllocationPercentages", wireType)
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
			v := &MintAllocationPercentages{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.AsType = &MsgUpdateParam_AsMintAllocationPercentages{v}
			iNdEx = postIndex
		case 4:
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
		case 5:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field AsFloat", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint64(encoding_binary.LittleEndian.Uint64(dAtA[iNdEx:]))
			iNdEx += 8
			m.AsType = &MsgUpdateParam_AsFloat{float64(math.Float64frombits(v))}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AsMintEqualsBurnClaimDistribution", wireType)
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
			v := &MintEqualsBurnClaimDistribution{}
			if err := v.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			m.AsType = &MsgUpdateParam_AsMintEqualsBurnClaimDistribution{v}
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
