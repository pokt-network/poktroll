// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: poktroll/proof/types.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
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
	return fileDescriptor_b75ef15dfd4d6998, []int{0}
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
	return fileDescriptor_b75ef15dfd4d6998, []int{1}
}

type Proof struct {
	// Address of the supplier's operator that submitted this proof.
	SupplierOperatorAddress string `protobuf:"bytes,1,opt,name=supplier_operator_address,json=supplierOperatorAddress,proto3" json:"supplier_operator_address,omitempty"`
	// The session header of the session that this claim is for.
	SessionHeader *types.SessionHeader `protobuf:"bytes,2,opt,name=session_header,json=sessionHeader,proto3" json:"session_header,omitempty"`
	// The serialized SMST proof from the `#ClosestProof()` method.
	ClosestMerkleProof []byte `protobuf:"bytes,3,opt,name=closest_merkle_proof,json=closestMerkleProof,proto3" json:"closest_merkle_proof,omitempty"`
}

func (m *Proof) Reset()         { *m = Proof{} }
func (m *Proof) String() string { return proto.CompactTextString(m) }
func (*Proof) ProtoMessage()    {}
func (*Proof) Descriptor() ([]byte, []int) {
	return fileDescriptor_b75ef15dfd4d6998, []int{0}
}
func (m *Proof) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Proof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Proof.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
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

// Claim is the serialized object stored on-chain for claims pending to be proven
type Claim struct {
	SupplierOperatorAddress string `protobuf:"bytes,1,opt,name=supplier_operator_address,json=supplierOperatorAddress,proto3" json:"supplier_operator_address,omitempty"`
	// The session header of the session that this claim is for.
	SessionHeader *types.SessionHeader `protobuf:"bytes,2,opt,name=session_header,json=sessionHeader,proto3" json:"session_header,omitempty"`
	// Root hash returned from smt.SMST#Root().
	RootHash []byte `protobuf:"bytes,3,opt,name=root_hash,json=rootHash,proto3" json:"root_hash,omitempty"`
}

func (m *Claim) Reset()         { *m = Claim{} }
func (m *Claim) String() string { return proto.CompactTextString(m) }
func (*Claim) ProtoMessage()    {}
func (*Claim) Descriptor() ([]byte, []int) {
	return fileDescriptor_b75ef15dfd4d6998, []int{1}
}
func (m *Claim) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Claim) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Claim.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
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

func init() {
	proto.RegisterEnum("poktroll.proof.ProofRequirementReason", ProofRequirementReason_name, ProofRequirementReason_value)
	proto.RegisterEnum("poktroll.proof.ClaimProofStage", ClaimProofStage_name, ClaimProofStage_value)
	proto.RegisterType((*Proof)(nil), "poktroll.proof.Proof")
	proto.RegisterType((*Claim)(nil), "poktroll.proof.Claim")
}

func init() { proto.RegisterFile("poktroll/proof/types.proto", fileDescriptor_b75ef15dfd4d6998) }

var fileDescriptor_b75ef15dfd4d6998 = []byte{
	// 439 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x92, 0x4f, 0x6e, 0xd3, 0x40,
	0x14, 0xc6, 0x33, 0xa9, 0x5a, 0xc8, 0xb4, 0x29, 0x66, 0x54, 0x41, 0x1a, 0x90, 0x89, 0xba, 0x8a,
	0x2a, 0xd5, 0x46, 0x70, 0x82, 0xfc, 0x31, 0xb2, 0x25, 0xb7, 0x0e, 0x63, 0x83, 0x10, 0x1b, 0x6b,
	0x9a, 0x0c, 0xb1, 0x15, 0xdb, 0xcf, 0xcc, 0x4c, 0x04, 0xdc, 0x82, 0xc3, 0x70, 0x02, 0x56, 0x2c,
	0x2b, 0x36, 0xb0, 0x44, 0xc9, 0x45, 0x90, 0x27, 0x4e, 0x94, 0x23, 0x74, 0x35, 0x7a, 0xf3, 0x7b,
	0xef, 0x7b, 0xfa, 0x9e, 0x3e, 0xdc, 0x2d, 0x61, 0xa1, 0x04, 0x64, 0x99, 0x5d, 0x0a, 0x80, 0x4f,
	0xb6, 0xfa, 0x56, 0x72, 0x69, 0x95, 0x02, 0x14, 0x90, 0xd3, 0x2d, 0xb3, 0x34, 0xeb, 0x9e, 0x4f,
	0x41, 0xe6, 0x20, 0x63, 0x4d, 0xed, 0x4d, 0xb1, 0x69, 0xed, 0x3e, 0xdf, 0xc9, 0x48, 0x2e, 0x65,
	0x0a, 0xc5, 0xbe, 0xd0, 0xc5, 0x1f, 0x84, 0x0f, 0x27, 0x95, 0x04, 0x89, 0xf0, 0xb9, 0x5c, 0x96,
	0x65, 0x96, 0x72, 0x11, 0x43, 0xc9, 0x05, 0x53, 0x20, 0x62, 0x36, 0x9b, 0x09, 0x2e, 0x65, 0x07,
	0xf5, 0x50, 0xbf, 0x35, 0xec, 0xfc, 0xfe, 0x71, 0x75, 0x56, 0x8b, 0x0f, 0x36, 0x24, 0x54, 0x22,
	0x2d, 0xe6, 0xf4, 0xe9, 0x76, 0x34, 0xa8, 0x27, 0x6b, 0x4c, 0xde, 0xe0, 0xd3, 0x7a, 0x6d, 0x9c,
	0x70, 0x36, 0xe3, 0xa2, 0xd3, 0xec, 0xa1, 0xfe, 0xf1, 0xab, 0x17, 0xd6, 0xce, 0x41, 0xcd, 0xad,
	0x70, 0xf3, 0xba, 0xba, 0x8d, 0xb6, 0xe5, 0x7e, 0x49, 0x5e, 0xe2, 0xb3, 0x69, 0x06, 0x92, 0x4b,
	0x15, 0xe7, 0x5c, 0x2c, 0x32, 0x1e, 0x6b, 0xe3, 0x9d, 0x83, 0x1e, 0xea, 0x9f, 0x50, 0x52, 0xb3,
	0x6b, 0x8d, 0xb4, 0x9f, 0x8b, 0x9f, 0x08, 0x1f, 0x8e, 0x32, 0x96, 0xe6, 0xf7, 0xdc, 0xd9, 0x33,
	0xdc, 0x12, 0x00, 0x2a, 0x4e, 0x98, 0x4c, 0x6a, 0x3b, 0x0f, 0xab, 0x0f, 0x97, 0xc9, 0xe4, 0xd2,
	0xc7, 0x4f, 0xb4, 0x1b, 0xca, 0x3f, 0x2f, 0x53, 0xc1, 0x73, 0x5e, 0x28, 0xca, 0x99, 0x84, 0x82,
	0x18, 0xf8, 0xe4, 0x26, 0x88, 0x62, 0xea, 0xbc, 0x7d, 0xe7, 0x51, 0x67, 0x6c, 0x34, 0xc8, 0x63,
	0xdc, 0x9e, 0xd0, 0x60, 0x38, 0x18, 0x7a, 0xbe, 0x17, 0x46, 0xde, 0xc8, 0x40, 0xa4, 0x8d, 0x5b,
	0x91, 0x4b, 0x9d, 0xd0, 0x0d, 0xfc, 0xb1, 0xd1, 0xbc, 0x1c, 0xe3, 0x47, 0xfa, 0x22, 0x5a, 0x32,
	0x54, 0x6c, 0xce, 0xc9, 0x31, 0x7e, 0x30, 0xf2, 0x07, 0xde, 0xb5, 0x56, 0xc0, 0xf8, 0x68, 0x42,
	0x83, 0xf7, 0xce, 0x8d, 0x81, 0x2a, 0x10, 0x3a, 0x51, 0xe4, 0x3b, 0x63, 0xa3, 0x59, 0x15, 0xce,
	0x87, 0x89, 0xde, 0x73, 0x30, 0x74, 0x7f, 0xad, 0x4c, 0x74, 0xb7, 0x32, 0xd1, 0xbf, 0x95, 0x89,
	0xbe, 0xaf, 0xcd, 0xc6, 0xdd, 0xda, 0x6c, 0xfc, 0x5d, 0x9b, 0x8d, 0x8f, 0xd6, 0x3c, 0x55, 0xc9,
	0xf2, 0xd6, 0x9a, 0x42, 0x6e, 0x57, 0x47, 0xb8, 0x2a, 0xb8, 0xfa, 0x02, 0x62, 0x61, 0xef, 0x22,
	0xf8, 0x75, 0x3f, 0xcb, 0xb7, 0x47, 0x3a, 0x83, 0xaf, 0xff, 0x07, 0x00, 0x00, 0xff, 0xff, 0xa3,
	0xc2, 0x94, 0xb8, 0xea, 0x02, 0x00, 0x00,
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