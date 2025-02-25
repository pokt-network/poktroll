// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: poktroll/migration/event.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	_ "github.com/pokt-network/poktroll/x/shared/types"
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

// EventImportMorseClaimableAccounts is emitted when the MorseClaimableAccounts are created on-chain.
type EventImportMorseClaimableAccounts struct {
	// The height (on Shannon) at which the MorseAccountState was created on-chain.
	CreatedAtHeight int64 `protobuf:"varint,1,opt,name=created_at_height,json=createdAtHeight,proto3" json:"created_at_height"`
	// The onchain computed sha256 hash of the entire MorseAccountState containing the MorseClaimableAccounts which were imported.
	MorseAccountStateHash []byte `protobuf:"bytes,2,opt,name=morse_account_state_hash,json=morseAccountStateHash,proto3" json:"morse_account_state_hash"`
	// Number of claimable accounts (EOAs) collected from Morse state export
	// NOTE: Account balances include consolidated application and supplier actor stakes
	NumAccounts uint64 `protobuf:"varint,3,opt,name=num_accounts,json=numAccounts,proto3" json:"num_accounts"`
}

func (m *EventImportMorseClaimableAccounts) Reset()         { *m = EventImportMorseClaimableAccounts{} }
func (m *EventImportMorseClaimableAccounts) String() string { return proto.CompactTextString(m) }
func (*EventImportMorseClaimableAccounts) ProtoMessage()    {}
func (*EventImportMorseClaimableAccounts) Descriptor() ([]byte, []int) {
	return fileDescriptor_d5b0bc9ed37905e1, []int{0}
}
func (m *EventImportMorseClaimableAccounts) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventImportMorseClaimableAccounts) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *EventImportMorseClaimableAccounts) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventImportMorseClaimableAccounts.Merge(m, src)
}
func (m *EventImportMorseClaimableAccounts) XXX_Size() int {
	return m.Size()
}
func (m *EventImportMorseClaimableAccounts) XXX_DiscardUnknown() {
	xxx_messageInfo_EventImportMorseClaimableAccounts.DiscardUnknown(m)
}

var xxx_messageInfo_EventImportMorseClaimableAccounts proto.InternalMessageInfo

func (m *EventImportMorseClaimableAccounts) GetCreatedAtHeight() int64 {
	if m != nil {
		return m.CreatedAtHeight
	}
	return 0
}

func (m *EventImportMorseClaimableAccounts) GetMorseAccountStateHash() []byte {
	if m != nil {
		return m.MorseAccountStateHash
	}
	return nil
}

func (m *EventImportMorseClaimableAccounts) GetNumAccounts() uint64 {
	if m != nil {
		return m.NumAccounts
	}
	return 0
}

func init() {
	proto.RegisterType((*EventImportMorseClaimableAccounts)(nil), "poktroll.migration.EventImportMorseClaimableAccounts")
}

func init() { proto.RegisterFile("poktroll/migration/event.proto", fileDescriptor_d5b0bc9ed37905e1) }

var fileDescriptor_d5b0bc9ed37905e1 = []byte{
	// 360 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0xcd, 0x8e, 0xd3, 0x30,
	0x14, 0x85, 0x6b, 0x8a, 0x58, 0x84, 0x4a, 0x40, 0x44, 0xa5, 0x50, 0x81, 0x5b, 0x58, 0xa0, 0x6e,
	0xa8, 0x55, 0xfa, 0x04, 0x0d, 0x42, 0x2a, 0x0b, 0x24, 0x54, 0xc4, 0x86, 0x4d, 0xe4, 0xb8, 0x56,
	0x1c, 0x35, 0xf6, 0x8d, 0xec, 0x9b, 0x02, 0x6f, 0xc1, 0x63, 0xb1, 0xec, 0xb2, 0xab, 0x0a, 0xa5,
	0xbb, 0x2e, 0xe7, 0x09, 0x46, 0xf9, 0xab, 0x66, 0x34, 0x33, 0x3b, 0xfb, 0x7c, 0xc7, 0xd7, 0xe7,
	0x1e, 0x8f, 0xe6, 0xb0, 0x45, 0x0b, 0x59, 0xc6, 0x74, 0x9a, 0x58, 0x8e, 0x29, 0x18, 0x26, 0x77,
	0xd2, 0xe0, 0x2c, 0xb7, 0x80, 0xe0, 0xfb, 0x1d, 0x9f, 0x5d, 0xf8, 0xe8, 0x95, 0x00, 0xa7, 0xc1,
	0x45, 0xb5, 0x83, 0x35, 0x97, 0xc6, 0x3e, 0x7a, 0x99, 0x40, 0x02, 0x8d, 0x5e, 0x9d, 0x5a, 0x95,
	0x36, 0x1e, 0x16, 0x73, 0x27, 0xd9, 0x6e, 0x1e, 0x4b, 0xe4, 0x73, 0x26, 0x20, 0x35, 0x2d, 0x7f,
	0x73, 0x09, 0xe1, 0x14, 0xb7, 0x72, 0xc3, 0x9c, 0xb4, 0xbb, 0x54, 0xc8, 0x16, 0xbf, 0xbf, 0x27,
	0xa3, 0x06, 0xeb, 0x64, 0x04, 0x46, 0x28, 0xde, 0x8d, 0x79, 0x77, 0x45, 0xbc, 0xb7, 0x9f, 0xab,
	0xec, 0x5f, 0x74, 0x0e, 0x16, 0xbf, 0x56, 0x96, 0x4f, 0x19, 0x4f, 0x35, 0x8f, 0x33, 0xb9, 0x14,
	0x02, 0x0a, 0x83, 0xce, 0x5f, 0x7a, 0x2f, 0x84, 0x95, 0x1c, 0xe5, 0x26, 0xe2, 0x18, 0x29, 0x99,
	0x26, 0x0a, 0x03, 0x32, 0x21, 0xd3, 0x7e, 0x38, 0x3c, 0x1f, 0xc7, 0x77, 0xe1, 0xfa, 0x59, 0x2b,
	0x2d, 0x71, 0x55, 0x0b, 0xfe, 0x0f, 0x2f, 0x68, 0xfe, 0xe7, 0xcd, 0xd0, 0xc8, 0x21, 0x47, 0x19,
	0x29, 0xee, 0x54, 0xf0, 0x68, 0x42, 0xa6, 0x83, 0xf0, 0xf5, 0xf9, 0x38, 0x7e, 0xd0, 0xb3, 0x1e,
	0xd6, 0xa4, 0x4d, 0xf4, 0xbd, 0xd2, 0x57, 0xdc, 0x29, 0x7f, 0xe1, 0x0d, 0x4c, 0xa1, 0xbb, 0x07,
	0x2e, 0xe8, 0x4f, 0xc8, 0xf4, 0x71, 0xf8, 0xfc, 0x7c, 0x1c, 0xdf, 0xd2, 0xd7, 0x4f, 0x4d, 0xa1,
	0xbb, 0x75, 0xc2, 0x6f, 0xff, 0x4a, 0x4a, 0xf6, 0x25, 0x25, 0x87, 0x92, 0x92, 0xff, 0x25, 0x25,
	0x7f, 0x4f, 0xb4, 0xb7, 0x3f, 0xd1, 0xde, 0xe1, 0x44, 0x7b, 0x3f, 0x3f, 0x26, 0x29, 0xaa, 0x22,
	0x9e, 0x09, 0xd0, 0xac, 0x6a, 0xf1, 0x83, 0x91, 0xf8, 0x0b, 0xec, 0x96, 0x5d, 0x2a, 0xfd, 0x7d,
	0xa3, 0x54, 0xfc, 0x93, 0x4b, 0x17, 0x3f, 0xa9, 0xdb, 0x5c, 0x5c, 0x07, 0x00, 0x00, 0xff, 0xff,
	0xe7, 0xcb, 0xa5, 0x40, 0x1b, 0x02, 0x00, 0x00,
}

func (m *EventImportMorseClaimableAccounts) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventImportMorseClaimableAccounts) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventImportMorseClaimableAccounts) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.NumAccounts != 0 {
		i = encodeVarintEvent(dAtA, i, uint64(m.NumAccounts))
		i--
		dAtA[i] = 0x18
	}
	if len(m.MorseAccountStateHash) > 0 {
		i -= len(m.MorseAccountStateHash)
		copy(dAtA[i:], m.MorseAccountStateHash)
		i = encodeVarintEvent(dAtA, i, uint64(len(m.MorseAccountStateHash)))
		i--
		dAtA[i] = 0x12
	}
	if m.CreatedAtHeight != 0 {
		i = encodeVarintEvent(dAtA, i, uint64(m.CreatedAtHeight))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintEvent(dAtA []byte, offset int, v uint64) int {
	offset -= sovEvent(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *EventImportMorseClaimableAccounts) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.CreatedAtHeight != 0 {
		n += 1 + sovEvent(uint64(m.CreatedAtHeight))
	}
	l = len(m.MorseAccountStateHash)
	if l > 0 {
		n += 1 + l + sovEvent(uint64(l))
	}
	if m.NumAccounts != 0 {
		n += 1 + sovEvent(uint64(m.NumAccounts))
	}
	return n
}

func sovEvent(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEvent(x uint64) (n int) {
	return sovEvent(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *EventImportMorseClaimableAccounts) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvent
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
			return fmt.Errorf("proto: EventImportMorseClaimableAccounts: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventImportMorseClaimableAccounts: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedAtHeight", wireType)
			}
			m.CreatedAtHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreatedAtHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MorseAccountStateHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
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
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MorseAccountStateHash = append(m.MorseAccountStateHash[:0], dAtA[iNdEx:postIndex]...)
			if m.MorseAccountStateHash == nil {
				m.MorseAccountStateHash = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NumAccounts", wireType)
			}
			m.NumAccounts = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NumAccounts |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipEvent(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEvent
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
func skipEvent(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEvent
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
					return 0, ErrIntOverflowEvent
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
					return 0, ErrIntOverflowEvent
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
				return 0, ErrInvalidLengthEvent
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEvent
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEvent
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEvent        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEvent          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEvent = fmt.Errorf("proto: unexpected end of group")
)
