// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pocket/proof/types.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	types "github.com/pokt-network/poktroll/x/session/types"
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

type ProofRequirementReason int32

const (
	ProofRequirementReason_NOT_REQUIRED  ProofRequirementReason = 0
	ProofRequirementReason_PROBABILISTIC ProofRequirementReason = 1
	ProofRequirementReason_THRESHOLD     ProofRequirementReason = 2
)

var ProofRequirementReason_name = map[int32]string{
	0: "NOT_REQUIRED",
	1: "PROBABILISTIC",
	2: "THRESHOLD",
}

var ProofRequirementReason_value = map[string]int32{
	"NOT_REQUIRED":  0,
	"PROBABILISTIC": 1,
	"THRESHOLD":     2,
}

func (x ProofRequirementReason) String() string {
	return proto.EnumName(ProofRequirementReason_name, int32(x))
}

func (ProofRequirementReason) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{0}
}

type ClaimProofStage int32

const (
	ClaimProofStage_CLAIMED ClaimProofStage = 0
	ClaimProofStage_PROVEN  ClaimProofStage = 1
	ClaimProofStage_SETTLED ClaimProofStage = 2
	ClaimProofStage_EXPIRED ClaimProofStage = 3
)

var ClaimProofStage_name = map[int32]string{
	0: "CLAIMED",
	1: "PROVEN",
	2: "SETTLED",
	3: "EXPIRED",
}

var ClaimProofStage_value = map[string]int32{
	"CLAIMED": 0,
	"PROVEN":  1,
	"SETTLED": 2,
	"EXPIRED": 3,
}

func (x ClaimProofStage) String() string {
	return proto.EnumName(ClaimProofStage_name, int32(x))
}

func (ClaimProofStage) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{1}
}

// Status of proof validation for a claim
// Default is PENDING_VALIDATION regardless of proof requirement
type ClaimProofStatus int32

const (
	ClaimProofStatus_PENDING_VALIDATION ClaimProofStatus = 0
	ClaimProofStatus_VALIDATED          ClaimProofStatus = 1
	ClaimProofStatus_INVALID            ClaimProofStatus = 2
)

var ClaimProofStatus_name = map[int32]string{
	0: "PENDING_VALIDATION",
	1: "VALIDATED",
	2: "INVALID",
}

var ClaimProofStatus_value = map[string]int32{
	"PENDING_VALIDATION": 0,
	"VALIDATED":          1,
	"INVALID":            2,
}

func (x ClaimProofStatus) String() string {
	return proto.EnumName(ClaimProofStatus_name, int32(x))
}

func (ClaimProofStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{2}
}

type Proof struct {
	// Address of the supplier's operator that submitted this proof.
	SupplierOperatorAddress string `protobuf:"bytes,1,opt,name=supplier_operator_address,json=supplierOperatorAddress,proto3" json:"supplier_operator_address,omitempty"`
	// The session header of the session that this claim is for.
	SessionHeader *types.SessionHeader `protobuf:"bytes,2,opt,name=session_header,json=sessionHeader,proto3" json:"session_header,omitempty"`
	// The serialized SMST compacted proof from the `#ClosestProof()` method.
	ClosestMerkleProof []byte `protobuf:"bytes,3,opt,name=closest_merkle_proof,json=closestMerkleProof,proto3" json:"closest_merkle_proof,omitempty"`
}

func (m *Proof) Reset()         { *m = Proof{} }
func (m *Proof) String() string { return proto.CompactTextString(m) }
func (*Proof) ProtoMessage()    {}
func (*Proof) Descriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{0}
}
func (m *Proof) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Proof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Proof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Proof.Merge(m, src)
}
func (m *Proof) XXX_Size() int {
	return m.Size()
}
func (m *Proof) XXX_DiscardUnknown() {
	xxx_messageInfo_Proof.DiscardUnknown(m)
}

var xxx_messageInfo_Proof proto.InternalMessageInfo

func (m *Proof) GetSupplierOperatorAddress() string {
	if m != nil {
		return m.SupplierOperatorAddress
	}
	return ""
}

func (m *Proof) GetSessionHeader() *types.SessionHeader {
	if m != nil {
		return m.SessionHeader
	}
	return nil
}

func (m *Proof) GetClosestMerkleProof() []byte {
	if m != nil {
		return m.ClosestMerkleProof
	}
	return nil
}

// Claim is the serialized object stored onchain for claims pending to be proven
type Claim struct {
	// Address of the supplier's operator that submitted this claim.
	SupplierOperatorAddress string `protobuf:"bytes,1,opt,name=supplier_operator_address,json=supplierOperatorAddress,proto3" json:"supplier_operator_address,omitempty"`
	// Session header this claim is for.
	SessionHeader *types.SessionHeader `protobuf:"bytes,2,opt,name=session_header,json=sessionHeader,proto3" json:"session_header,omitempty"`
	// Root hash from smt.SMST#Root().
	// TODO_UP_NEXT(@bryanchriswhite, #1497): Dehydrate the claim's root hash from onchain events.
	RootHash []byte `protobuf:"bytes,3,opt,name=root_hash,json=rootHash,proto3" json:"root_hash,omitempty"`
	// Important: This field MUST only be set by proofKeeper#EnsureValidProofSignaturesAndClosestPath
	ProofValidationStatus ClaimProofStatus `protobuf:"varint,4,opt,name=proof_validation_status,json=proofValidationStatus,proto3,enum=pocket.proof.ClaimProofStatus" json:"proof_validation_status,omitempty"`
}

func (m *Claim) Reset()         { *m = Claim{} }
func (m *Claim) String() string { return proto.CompactTextString(m) }
func (*Claim) ProtoMessage()    {}
func (*Claim) Descriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{1}
}
func (m *Claim) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Claim) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Claim) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Claim.Merge(m, src)
}
func (m *Claim) XXX_Size() int {
	return m.Size()
}
func (m *Claim) XXX_DiscardUnknown() {
	xxx_messageInfo_Claim.DiscardUnknown(m)
}

var xxx_messageInfo_Claim proto.InternalMessageInfo

func (m *Claim) GetSupplierOperatorAddress() string {
	if m != nil {
		return m.SupplierOperatorAddress
	}
	return ""
}

func (m *Claim) GetSessionHeader() *types.SessionHeader {
	if m != nil {
		return m.SessionHeader
	}
	return nil
}

func (m *Claim) GetRootHash() []byte {
	if m != nil {
		return m.RootHash
	}
	return nil
}

func (m *Claim) GetProofValidationStatus() ClaimProofStatus {
	if m != nil {
		return m.ProofValidationStatus
	}
	return ClaimProofStatus_PENDING_VALIDATION
}

// SessionSMT is the serializable session's SMST used to persist the session's
// state offchain by the RelayMiner.
// It is not used for any onchain logic.
type SessionSMT struct {
	SessionHeader           *types.SessionHeader `protobuf:"bytes,1,opt,name=session_header,json=sessionHeader,proto3" json:"session_header,omitempty"`
	SupplierOperatorAddress string               `protobuf:"bytes,2,opt,name=supplier_operator_address,json=supplierOperatorAddress,proto3" json:"supplier_operator_address,omitempty"`
	SmtRoot                 []byte               `protobuf:"bytes,3,opt,name=smt_root,json=smtRoot,proto3" json:"smt_root,omitempty"`
}

func (m *SessionSMT) Reset()         { *m = SessionSMT{} }
func (m *SessionSMT) String() string { return proto.CompactTextString(m) }
func (*SessionSMT) ProtoMessage()    {}
func (*SessionSMT) Descriptor() ([]byte, []int) {
	return fileDescriptor_cdde56dba22629df, []int{2}
}
func (m *SessionSMT) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SessionSMT) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *SessionSMT) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SessionSMT.Merge(m, src)
}
func (m *SessionSMT) XXX_Size() int {
	return m.Size()
}
func (m *SessionSMT) XXX_DiscardUnknown() {
	xxx_messageInfo_SessionSMT.DiscardUnknown(m)
}

var xxx_messageInfo_SessionSMT proto.InternalMessageInfo

func (m *SessionSMT) GetSessionHeader() *types.SessionHeader {
	if m != nil {
		return m.SessionHeader
	}
	return nil
}

func (m *SessionSMT) GetSupplierOperatorAddress() string {
	if m != nil {
		return m.SupplierOperatorAddress
	}
	return ""
}

func (m *SessionSMT) GetSmtRoot() []byte {
	if m != nil {
		return m.SmtRoot
	}
	return nil
}

func init() {
	proto.RegisterEnum("pocket.proof.ProofRequirementReason", ProofRequirementReason_name, ProofRequirementReason_value)
	proto.RegisterEnum("pocket.proof.ClaimProofStage", ClaimProofStage_name, ClaimProofStage_value)
	proto.RegisterEnum("pocket.proof.ClaimProofStatus", ClaimProofStatus_name, ClaimProofStatus_value)
	proto.RegisterType((*Proof)(nil), "pocket.proof.Proof")
	proto.RegisterType((*Claim)(nil), "pocket.proof.Claim")
	proto.RegisterType((*SessionSMT)(nil), "pocket.proof.SessionSMT")
}

func init() { proto.RegisterFile("pocket/proof/types.proto", fileDescriptor_cdde56dba22629df) }

var fileDescriptor_cdde56dba22629df = []byte{
	// 570 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x94, 0xcb, 0x6e, 0xd3, 0x40,
	0x14, 0x86, 0x33, 0x2e, 0xbd, 0x4d, 0x2f, 0x98, 0x51, 0x69, 0xdd, 0x20, 0xac, 0xa8, 0xab, 0xa8,
	0x52, 0x1d, 0x54, 0x9e, 0x20, 0x89, 0x0d, 0xb1, 0xe4, 0xd8, 0x61, 0x6c, 0x22, 0xc4, 0xc6, 0x72,
	0x93, 0x21, 0xb1, 0x62, 0x67, 0xcc, 0xcc, 0x84, 0xcb, 0x5b, 0xb0, 0xe4, 0x41, 0x78, 0x03, 0x36,
	0x2c, 0x2b, 0x24, 0xa4, 0x2e, 0x51, 0xf2, 0x22, 0xc8, 0xe3, 0x01, 0x05, 0x84, 0x58, 0x74, 0xc5,
	0xca, 0x3e, 0xe7, 0x9f, 0x73, 0xf9, 0x7e, 0x5f, 0xa0, 0x51, 0xd0, 0xd1, 0x8c, 0x88, 0x56, 0xc1,
	0x28, 0x7d, 0xd5, 0x12, 0xef, 0x0b, 0xc2, 0xad, 0x82, 0x51, 0x41, 0xd1, 0x7e, 0xa5, 0x58, 0x52,
	0xa9, 0x9f, 0x8e, 0x28, 0xcf, 0x29, 0x8f, 0xa5, 0xd6, 0xaa, 0x82, 0xea, 0x60, 0xbd, 0xae, 0x5a,
	0x70, 0xc2, 0x79, 0x4a, 0xe7, 0xeb, 0x4d, 0xea, 0x47, 0x13, 0x3a, 0xa1, 0x55, 0x4d, 0x79, 0x57,
	0x65, 0xcf, 0xbe, 0x01, 0xb8, 0x39, 0x28, 0xdb, 0xa2, 0x08, 0x9e, 0xf2, 0x45, 0x51, 0x64, 0x29,
	0x61, 0x31, 0x2d, 0x08, 0x4b, 0x04, 0x65, 0x71, 0x32, 0x1e, 0x33, 0xc2, 0xb9, 0x01, 0x1a, 0xa0,
	0xb9, 0xdb, 0x31, 0xbe, 0x7e, 0xba, 0x38, 0x52, 0x03, 0xdb, 0x95, 0x12, 0x0a, 0x96, 0xce, 0x27,
	0xf8, 0xe4, 0x67, 0x69, 0xa0, 0x2a, 0x95, 0x8c, 0x6c, 0x78, 0xa8, 0x96, 0x89, 0xa7, 0x24, 0x19,
	0x13, 0x66, 0x68, 0x0d, 0xd0, 0xdc, 0xbb, 0x7c, 0x68, 0x29, 0x26, 0xa5, 0x5a, 0x61, 0x75, 0xed,
	0xc9, 0x43, 0xf8, 0x80, 0xaf, 0x87, 0xe8, 0x11, 0x3c, 0x1a, 0x65, 0x94, 0x13, 0x2e, 0xe2, 0x9c,
	0xb0, 0x59, 0x46, 0x62, 0x69, 0x85, 0xb1, 0xd1, 0x00, 0xcd, 0x7d, 0x8c, 0x94, 0xd6, 0x97, 0x92,
	0xa4, 0x39, 0xfb, 0xa8, 0xc1, 0xcd, 0x6e, 0x96, 0xa4, 0xf9, 0x7f, 0xcd, 0xf5, 0x00, 0xee, 0x32,
	0x4a, 0x45, 0x3c, 0x4d, 0xf8, 0x54, 0xc1, 0xec, 0x94, 0x89, 0x5e, 0xc2, 0xa7, 0x68, 0x08, 0x4f,
	0x24, 0x65, 0xfc, 0x26, 0xc9, 0xd2, 0x71, 0x22, 0xca, 0x59, 0x5c, 0x24, 0x62, 0xc1, 0x8d, 0x3b,
	0x0d, 0xd0, 0x3c, 0xbc, 0x34, 0xad, 0xf5, 0xf7, 0xc2, 0x92, 0xb8, 0x92, 0x3e, 0x94, 0xa7, 0xf0,
	0x7d, 0x99, 0x1f, 0xfe, 0xaa, 0xae, 0xd2, 0x67, 0x9f, 0x01, 0x84, 0x6a, 0xab, 0xb0, 0x1f, 0xfd,
	0x85, 0x04, 0xdc, 0x82, 0xe4, 0x9f, 0x2e, 0x6b, 0xb7, 0x75, 0xf9, 0x14, 0xee, 0xf0, 0x5c, 0xc4,
	0xa5, 0x25, 0xca, 0x9e, 0x6d, 0x9e, 0x0b, 0x4c, 0xa9, 0x38, 0xf7, 0xe0, 0xb1, 0x64, 0xc5, 0xe4,
	0xf5, 0x22, 0x65, 0x24, 0x27, 0x73, 0x81, 0x49, 0xc2, 0xe9, 0x1c, 0xe9, 0x70, 0xdf, 0x0f, 0xa2,
	0x18, 0x3b, 0xcf, 0x9e, 0xbb, 0xd8, 0xb1, 0xf5, 0x1a, 0xba, 0x07, 0x0f, 0x06, 0x38, 0xe8, 0xb4,
	0x3b, 0xae, 0xe7, 0x86, 0x91, 0xdb, 0xd5, 0x01, 0x3a, 0x80, 0xbb, 0x51, 0x0f, 0x3b, 0x61, 0x2f,
	0xf0, 0x6c, 0x5d, 0x3b, 0xb7, 0xe1, 0xdd, 0xdf, 0xec, 0x9b, 0x10, 0xb4, 0x07, 0xb7, 0xbb, 0x5e,
	0xdb, 0xed, 0xcb, 0x0e, 0x10, 0x6e, 0x0d, 0x70, 0x30, 0x74, 0x7c, 0x1d, 0x94, 0x42, 0xe8, 0x44,
	0x91, 0xe7, 0xd8, 0xba, 0x56, 0x06, 0xce, 0x8b, 0x81, 0x9c, 0xb3, 0x71, 0xfe, 0x04, 0xea, 0x7f,
	0x3e, 0x04, 0x74, 0x0c, 0xd1, 0xc0, 0xf1, 0x6d, 0xd7, 0x7f, 0x1a, 0x0f, 0xdb, 0x9e, 0x6b, 0xb7,
	0x23, 0x37, 0xf0, 0xf5, 0x5a, 0xb9, 0x80, 0x8a, 0x1d, 0xbb, 0x6a, 0xea, 0xfa, 0x32, 0xa1, 0x6b,
	0x1d, 0xef, 0xcb, 0xd2, 0x04, 0xd7, 0x4b, 0x13, 0xdc, 0x2c, 0x4d, 0xf0, 0x7d, 0x69, 0x82, 0x0f,
	0x2b, 0xb3, 0x76, 0xbd, 0x32, 0x6b, 0x37, 0x2b, 0xb3, 0xf6, 0xd2, 0x9a, 0xa4, 0x62, 0xba, 0xb8,
	0xb2, 0x46, 0x34, 0x6f, 0x15, 0x74, 0x26, 0x2e, 0xe6, 0x44, 0xbc, 0xa5, 0x6c, 0x26, 0x03, 0x46,
	0xb3, 0xac, 0xf5, 0x6e, 0xfd, 0x1f, 0x72, 0xb5, 0x25, 0xbf, 0xf4, 0xc7, 0x3f, 0x02, 0x00, 0x00,
	0xff, 0xff, 0xf4, 0xd7, 0x7e, 0x74, 0x60, 0x04, 0x00, 0x00,
}

func (m *Proof) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Proof) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Proof) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ClosestMerkleProof) > 0 {
		i -= len(m.ClosestMerkleProof)
		copy(dAtA[i:], m.ClosestMerkleProof)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.ClosestMerkleProof)))
		i--
		dAtA[i] = 0x1a
	}
	if m.SessionHeader != nil {
		{
			size, err := m.SessionHeader.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTypes(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.SupplierOperatorAddress) > 0 {
		i -= len(m.SupplierOperatorAddress)
		copy(dAtA[i:], m.SupplierOperatorAddress)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.SupplierOperatorAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Claim) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Claim) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Claim) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ProofValidationStatus != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.ProofValidationStatus))
		i--
		dAtA[i] = 0x20
	}
	if len(m.RootHash) > 0 {
		i -= len(m.RootHash)
		copy(dAtA[i:], m.RootHash)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.RootHash)))
		i--
		dAtA[i] = 0x1a
	}
	if m.SessionHeader != nil {
		{
			size, err := m.SessionHeader.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTypes(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.SupplierOperatorAddress) > 0 {
		i -= len(m.SupplierOperatorAddress)
		copy(dAtA[i:], m.SupplierOperatorAddress)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.SupplierOperatorAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *SessionSMT) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SessionSMT) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SessionSMT) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.SmtRoot) > 0 {
		i -= len(m.SmtRoot)
		copy(dAtA[i:], m.SmtRoot)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.SmtRoot)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.SupplierOperatorAddress) > 0 {
		i -= len(m.SupplierOperatorAddress)
		copy(dAtA[i:], m.SupplierOperatorAddress)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.SupplierOperatorAddress)))
		i--
		dAtA[i] = 0x12
	}
	if m.SessionHeader != nil {
		{
			size, err := m.SessionHeader.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTypes(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypes(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypes(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Proof) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SupplierOperatorAddress)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.SessionHeader != nil {
		l = m.SessionHeader.Size()
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.ClosestMerkleProof)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	return n
}

func (m *Claim) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SupplierOperatorAddress)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.SessionHeader != nil {
		l = m.SessionHeader.Size()
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.RootHash)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.ProofValidationStatus != 0 {
		n += 1 + sovTypes(uint64(m.ProofValidationStatus))
	}
	return n
}

func (m *SessionSMT) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.SessionHeader != nil {
		l = m.SessionHeader.Size()
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.SupplierOperatorAddress)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.SmtRoot)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	return n
}

func sovTypes(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypes(x uint64) (n int) {
	return sovTypes(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Proof) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: Proof: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Proof: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SupplierOperatorAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SupplierOperatorAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SessionHeader", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.SessionHeader == nil {
				m.SessionHeader = &types.SessionHeader{}
			}
			if err := m.SessionHeader.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClosestMerkleProof", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClosestMerkleProof = append(m.ClosestMerkleProof[:0], dAtA[iNdEx:postIndex]...)
			if m.ClosestMerkleProof == nil {
				m.ClosestMerkleProof = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypes
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
func (m *Claim) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: Claim: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Claim: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SupplierOperatorAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SupplierOperatorAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SessionHeader", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.SessionHeader == nil {
				m.SessionHeader = &types.SessionHeader{}
			}
			if err := m.SessionHeader.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RootHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RootHash = append(m.RootHash[:0], dAtA[iNdEx:postIndex]...)
			if m.RootHash == nil {
				m.RootHash = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProofValidationStatus", wireType)
			}
			m.ProofValidationStatus = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProofValidationStatus |= ClaimProofStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypes
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
func (m *SessionSMT) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: SessionSMT: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SessionSMT: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SessionHeader", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.SessionHeader == nil {
				m.SessionHeader = &types.SessionHeader{}
			}
			if err := m.SessionHeader.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SupplierOperatorAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SupplierOperatorAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SmtRoot", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SmtRoot = append(m.SmtRoot[:0], dAtA[iNdEx:postIndex]...)
			if m.SmtRoot == nil {
				m.SmtRoot = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypes
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
func skipTypes(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
				return 0, ErrInvalidLengthTypes
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypes
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypes
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypes        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypes          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypes = fmt.Errorf("proto: unexpected end of group")
)
