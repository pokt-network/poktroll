// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: poktroll/migration/morse_onchain.proto

package types

import (
	crypto_ed25519 "crypto/ed25519"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

// MorseAccountState is the onchain representation of all account state to be migrated from Morse.
// It is NEVER persisted onchain but is a dependency of the MsgImportMorseClaimableAccount handler.
// It's main purpose is to expose the #GetHash() method for verifying the integrity of all MorseClaimableAccounts.
type MorseAccountState struct {
	Accounts []*MorseClaimableAccount `protobuf:"bytes,2,rep,name=accounts,proto3" json:"accounts" yaml:"accounts"`
}

func (m *MorseAccountState) Reset()         { *m = MorseAccountState{} }
func (m *MorseAccountState) String() string { return proto.CompactTextString(m) }
func (*MorseAccountState) ProtoMessage()    {}
func (*MorseAccountState) Descriptor() ([]byte, []int) {
	return fileDescriptor_e74ea76a959fdb61, []int{0}
}
func (m *MorseAccountState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MorseAccountState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *MorseAccountState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MorseAccountState.Merge(m, src)
}
func (m *MorseAccountState) XXX_Size() int {
	return m.Size()
}
func (m *MorseAccountState) XXX_DiscardUnknown() {
	xxx_messageInfo_MorseAccountState.DiscardUnknown(m)
}

var xxx_messageInfo_MorseAccountState proto.InternalMessageInfo

func (m *MorseAccountState) GetAccounts() []*MorseClaimableAccount {
	if m != nil {
		return m.Accounts
	}
	return nil
}

// MorseClaimableAccount is the onchain (persisted) representation of a Morse
// account which is claimable as part of the Morse -> Shannon migration.
// They are intended to be created during MorseAccountState import (see: MsgImportMorseClaimableAccount).
// It is created ONLY ONCE and NEVER deleted (per morse_src_address per network / re-genesis).
// It is updated ONLY ONCE, when it is claimed (per morse_src_address per network / re-genesis).
type MorseClaimableAccount struct {
	// The bech32-encoded address of the Shannon account to which the claimed balance will be minted.
	// This field is intended to remain empty until the account has been claimed.
	ShannonDestAddress string `protobuf:"bytes,1,opt,name=shannon_dest_address,json=shannonDestAddress,proto3" json:"shannon_dest_address"`
	// The hex-encoded address of the Morse account whose balance will be claimed.
	MorseSrcAddress string `protobuf:"bytes,2,opt,name=morse_src_address,json=morseSrcAddress,proto3" json:"morse_src_address"`
	// The ed25519 public key of the account.
	PublicKey crypto_ed25519.PublicKey `protobuf:"bytes,4,opt,name=public_key,json=publicKey,proto3,casttype=crypto/ed25519.PublicKey" json:"public_key,omitempty"`
	// The unstaked upokt tokens (i.e. account balance) available for claiming.
	UnstakedBalance types.Coin `protobuf:"bytes,5,opt,name=unstaked_balance,json=unstakedBalance,proto3" json:"unstaked_balance"`
	// The staked tokens associated with a supplier actor which corresponds to this account address.
	// DEV_NOTE: A few contextual notes related to Morse:
	// - A Supplier is called a Servicer or Node (not a full node) in Morse
	// - All Validators are Servicers, not all servicers are Validators
	// - Automatically, the top 100 staked Servicers are validator
	// - This only accounts for servicer stake balance transition
	// TODO_MAINNET(@Olshansk): Develop a strategy for bootstrapping validators in Shannon by working with the cosmos ecosystem
	SupplierStake types.Coin `protobuf:"bytes,6,opt,name=supplier_stake,json=supplierStake,proto3" json:"supplier_stake"`
	// The staked tokens associated with an application actor which corresponds to this account address.
	ApplicationStake types.Coin `protobuf:"bytes,7,opt,name=application_stake,json=applicationStake,proto3" json:"application_stake"`
	// The Shannon height at which the account was claimed.
	// This field is intended to remain empty until the account has been claimed.
	ClaimedAtHeight int64 `protobuf:"varint,8,opt,name=claimed_at_height,json=claimedAtHeight,proto3" json:"claimed_at_height" yaml:"claimed_at_height"`
}

func (m *MorseClaimableAccount) Reset()         { *m = MorseClaimableAccount{} }
func (m *MorseClaimableAccount) String() string { return proto.CompactTextString(m) }
func (*MorseClaimableAccount) ProtoMessage()    {}
func (*MorseClaimableAccount) Descriptor() ([]byte, []int) {
	return fileDescriptor_e74ea76a959fdb61, []int{1}
}
func (m *MorseClaimableAccount) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MorseClaimableAccount) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *MorseClaimableAccount) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MorseClaimableAccount.Merge(m, src)
}
func (m *MorseClaimableAccount) XXX_Size() int {
	return m.Size()
}
func (m *MorseClaimableAccount) XXX_DiscardUnknown() {
	xxx_messageInfo_MorseClaimableAccount.DiscardUnknown(m)
}

var xxx_messageInfo_MorseClaimableAccount proto.InternalMessageInfo

func (m *MorseClaimableAccount) GetShannonDestAddress() string {
	if m != nil {
		return m.ShannonDestAddress
	}
	return ""
}

func (m *MorseClaimableAccount) GetMorseSrcAddress() string {
	if m != nil {
		return m.MorseSrcAddress
	}
	return ""
}

func (m *MorseClaimableAccount) GetPublicKey() crypto_ed25519.PublicKey {
	if m != nil {
		return m.PublicKey
	}
	return nil
}

func (m *MorseClaimableAccount) GetUnstakedBalance() types.Coin {
	if m != nil {
		return m.UnstakedBalance
	}
	return types.Coin{}
}

func (m *MorseClaimableAccount) GetSupplierStake() types.Coin {
	if m != nil {
		return m.SupplierStake
	}
	return types.Coin{}
}

func (m *MorseClaimableAccount) GetApplicationStake() types.Coin {
	if m != nil {
		return m.ApplicationStake
	}
	return types.Coin{}
}

func (m *MorseClaimableAccount) GetClaimedAtHeight() int64 {
	if m != nil {
		return m.ClaimedAtHeight
	}
	return 0
}

func init() {
	proto.RegisterType((*MorseAccountState)(nil), "poktroll.migration.MorseAccountState")
	proto.RegisterType((*MorseClaimableAccount)(nil), "poktroll.migration.MorseClaimableAccount")
}

func init() {
	proto.RegisterFile("poktroll/migration/morse_onchain.proto", fileDescriptor_e74ea76a959fdb61)
}

var fileDescriptor_e74ea76a959fdb61 = []byte{
	// 552 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x53, 0x31, 0x6f, 0xd3, 0x40,
	0x18, 0x8d, 0xdb, 0x52, 0xda, 0x2b, 0x90, 0xc6, 0x4a, 0x91, 0x13, 0x21, 0x3b, 0xca, 0x80, 0xc2,
	0x50, 0x5b, 0x09, 0x64, 0x00, 0xa6, 0xb8, 0x0c, 0x48, 0x08, 0xa9, 0x72, 0x16, 0xc4, 0x80, 0x75,
	0x3e, 0x5f, 0x1d, 0x2b, 0xf6, 0x9d, 0x75, 0x77, 0x01, 0x22, 0xfe, 0x04, 0x3f, 0x86, 0x1f, 0x91,
	0xb1, 0x62, 0xea, 0x64, 0xa1, 0x64, 0xcb, 0xd8, 0x91, 0x05, 0x64, 0xfb, 0x6c, 0x55, 0x75, 0xa5,
	0x6e, 0xf7, 0xde, 0xf7, 0xde, 0xfb, 0xec, 0xbb, 0xef, 0x03, 0xcf, 0x13, 0x3a, 0x17, 0x8c, 0x46,
	0x91, 0x15, 0x87, 0x01, 0x83, 0x22, 0xa4, 0xc4, 0x8a, 0x29, 0xe3, 0xd8, 0xa5, 0x04, 0xcd, 0x60,
	0x48, 0xcc, 0x84, 0x51, 0x41, 0x55, 0xb5, 0xd4, 0x99, 0x95, 0xae, 0xdb, 0x41, 0x94, 0xc7, 0x94,
	0xbb, 0xb9, 0xc2, 0x2a, 0x40, 0x21, 0xef, 0xea, 0x05, 0xb2, 0x3c, 0xc8, 0xb1, 0xf5, 0x75, 0xe8,
	0x61, 0x01, 0x87, 0x16, 0xa2, 0x65, 0x5c, 0xb7, 0x1d, 0xd0, 0x80, 0x16, 0xbe, 0xec, 0x54, 0xb0,
	0xfd, 0x1f, 0xa0, 0xf5, 0x31, 0xeb, 0x3d, 0x41, 0x88, 0x2e, 0x88, 0x98, 0x0a, 0x28, 0xb0, 0x7a,
	0x01, 0x0e, 0x60, 0x81, 0xb9, 0xb6, 0xd3, 0xdb, 0x1d, 0x1c, 0x8d, 0x5e, 0x98, 0xf5, 0x8f, 0x31,
	0x73, 0xe3, 0x59, 0x04, 0xc3, 0x18, 0x7a, 0x51, 0x99, 0x60, 0x1b, 0xdb, 0xd4, 0xa8, 0xec, 0xd7,
	0xa9, 0xd1, 0x5c, 0xc2, 0x38, 0x7a, 0xd3, 0x2f, 0x99, 0xbe, 0x53, 0x15, 0xfb, 0xff, 0xf6, 0xc0,
	0xc9, 0x9d, 0x21, 0xea, 0x05, 0x68, 0xf3, 0x19, 0x24, 0x84, 0x12, 0xd7, 0xc7, 0x5c, 0xb8, 0xd0,
	0xf7, 0x19, 0xe6, 0x5c, 0x53, 0x7a, 0xca, 0xe0, 0xd0, 0x7e, 0xb5, 0x4a, 0x0d, 0x65, 0x9b, 0x1a,
	0x77, 0x6a, 0x7e, 0xff, 0x3a, 0x6d, 0xcb, 0x8b, 0x99, 0x14, 0xcc, 0x54, 0xb0, 0x90, 0x04, 0x8e,
	0x2a, 0xd5, 0xef, 0x30, 0x17, 0xb2, 0xa2, 0x4e, 0x40, 0xab, 0xb8, 0x7a, 0xce, 0x50, 0xd5, 0x64,
	0x27, 0x6f, 0x72, 0xb2, 0x4d, 0x8d, 0x7a, 0xd1, 0x69, 0xe6, 0xd4, 0x94, 0xa1, 0x32, 0xe2, 0x2d,
	0x00, 0xc9, 0xc2, 0x8b, 0x42, 0xe4, 0xce, 0xf1, 0x52, 0xdb, 0xeb, 0x29, 0x83, 0x47, 0xf6, 0xb3,
	0xbf, 0xa9, 0xa1, 0x21, 0xb6, 0x4c, 0x04, 0xb5, 0xb0, 0x3f, 0x1a, 0x8f, 0x87, 0xaf, 0xcd, 0xf3,
	0x5c, 0xf4, 0x01, 0x2f, 0x9d, 0xc3, 0xa4, 0x3c, 0xaa, 0x5f, 0xc0, 0xf1, 0x82, 0x70, 0x01, 0xe7,
	0xd8, 0x77, 0x3d, 0x18, 0x41, 0x82, 0xb0, 0xf6, 0xa0, 0xa7, 0x0c, 0x8e, 0x46, 0x1d, 0x53, 0xfe,
	0x44, 0xf6, 0x9e, 0xa6, 0x7c, 0x4f, 0xf3, 0x8c, 0x86, 0xc4, 0xd6, 0x56, 0xa9, 0xd1, 0xd8, 0xa6,
	0x46, 0xcd, 0xea, 0x34, 0x4b, 0xc6, 0x2e, 0x08, 0xf5, 0x13, 0x78, 0xc2, 0x17, 0x49, 0x12, 0x85,
	0x98, 0xb9, 0x79, 0x45, 0xdb, 0xbf, 0x2f, 0xfd, 0xa9, 0x4c, 0xbf, 0x65, 0x74, 0x1e, 0x97, 0x78,
	0x9a, 0x41, 0x15, 0x82, 0x16, 0xcc, 0x30, 0xca, 0x67, 0x41, 0x86, 0x3f, 0xbc, 0x2f, 0xbc, 0x23,
	0xc3, 0xeb, 0x5e, 0xe7, 0xf8, 0x06, 0x55, 0xb5, 0x40, 0xd9, 0x60, 0x60, 0xdf, 0x85, 0xc2, 0x9d,
	0xe1, 0x30, 0x98, 0x09, 0xed, 0xa0, 0xa7, 0x0c, 0x76, 0xed, 0xb1, 0x9c, 0x80, 0xba, 0xe0, 0x3a,
	0x35, 0xb4, 0x62, 0xe2, 0x6a, 0xa5, 0xbe, 0xd3, 0x94, 0xdc, 0x44, 0xbc, 0xcf, 0x19, 0xfb, 0x7c,
	0xb5, 0xd6, 0x95, 0xcb, 0xb5, 0xae, 0x5c, 0xad, 0x75, 0xe5, 0xcf, 0x5a, 0x57, 0x7e, 0x6e, 0xf4,
	0xc6, 0xe5, 0x46, 0x6f, 0x5c, 0x6d, 0xf4, 0xc6, 0xe7, 0x51, 0x10, 0x8a, 0xd9, 0xc2, 0x33, 0x11,
	0x8d, 0xad, 0x6c, 0xfe, 0x4f, 0x09, 0x16, 0xdf, 0x28, 0x9b, 0x5b, 0xd5, 0x06, 0x7f, 0xbf, 0xb1,
	0xc3, 0x62, 0x99, 0x60, 0xee, 0xed, 0xe7, 0x7b, 0xf5, 0xf2, 0x7f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x9e, 0x69, 0xe7, 0xf5, 0xe6, 0x03, 0x00, 0x00,
}

func (m *MorseAccountState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MorseAccountState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MorseAccountState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Accounts) > 0 {
		for iNdEx := len(m.Accounts) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Accounts[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintMorseOnchain(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	return len(dAtA) - i, nil
}

func (m *MorseClaimableAccount) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MorseClaimableAccount) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MorseClaimableAccount) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ClaimedAtHeight != 0 {
		i = encodeVarintMorseOnchain(dAtA, i, uint64(m.ClaimedAtHeight))
		i--
		dAtA[i] = 0x40
	}
	{
		size, err := m.ApplicationStake.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMorseOnchain(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x3a
	{
		size, err := m.SupplierStake.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMorseOnchain(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
	{
		size, err := m.UnstakedBalance.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMorseOnchain(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if len(m.PublicKey) > 0 {
		i -= len(m.PublicKey)
		copy(dAtA[i:], m.PublicKey)
		i = encodeVarintMorseOnchain(dAtA, i, uint64(len(m.PublicKey)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.MorseSrcAddress) > 0 {
		i -= len(m.MorseSrcAddress)
		copy(dAtA[i:], m.MorseSrcAddress)
		i = encodeVarintMorseOnchain(dAtA, i, uint64(len(m.MorseSrcAddress)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ShannonDestAddress) > 0 {
		i -= len(m.ShannonDestAddress)
		copy(dAtA[i:], m.ShannonDestAddress)
		i = encodeVarintMorseOnchain(dAtA, i, uint64(len(m.ShannonDestAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMorseOnchain(dAtA []byte, offset int, v uint64) int {
	offset -= sovMorseOnchain(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MorseAccountState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Accounts) > 0 {
		for _, e := range m.Accounts {
			l = e.Size()
			n += 1 + l + sovMorseOnchain(uint64(l))
		}
	}
	return n
}

func (m *MorseClaimableAccount) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ShannonDestAddress)
	if l > 0 {
		n += 1 + l + sovMorseOnchain(uint64(l))
	}
	l = len(m.MorseSrcAddress)
	if l > 0 {
		n += 1 + l + sovMorseOnchain(uint64(l))
	}
	l = len(m.PublicKey)
	if l > 0 {
		n += 1 + l + sovMorseOnchain(uint64(l))
	}
	l = m.UnstakedBalance.Size()
	n += 1 + l + sovMorseOnchain(uint64(l))
	l = m.SupplierStake.Size()
	n += 1 + l + sovMorseOnchain(uint64(l))
	l = m.ApplicationStake.Size()
	n += 1 + l + sovMorseOnchain(uint64(l))
	if m.ClaimedAtHeight != 0 {
		n += 1 + sovMorseOnchain(uint64(m.ClaimedAtHeight))
	}
	return n
}

func sovMorseOnchain(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMorseOnchain(x uint64) (n int) {
	return sovMorseOnchain(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MorseAccountState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMorseOnchain
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
			return fmt.Errorf("proto: MorseAccountState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MorseAccountState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Accounts", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Accounts = append(m.Accounts, &MorseClaimableAccount{})
			if err := m.Accounts[len(m.Accounts)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMorseOnchain(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMorseOnchain
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
func (m *MorseClaimableAccount) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMorseOnchain
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
			return fmt.Errorf("proto: MorseClaimableAccount: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MorseClaimableAccount: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ShannonDestAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ShannonDestAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MorseSrcAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MorseSrcAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PublicKey", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PublicKey = append(m.PublicKey[:0], dAtA[iNdEx:postIndex]...)
			if m.PublicKey == nil {
				m.PublicKey = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UnstakedBalance", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.UnstakedBalance.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SupplierStake", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SupplierStake.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ApplicationStake", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
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
				return ErrInvalidLengthMorseOnchain
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMorseOnchain
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ApplicationStake.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClaimedAtHeight", wireType)
			}
			m.ClaimedAtHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMorseOnchain
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ClaimedAtHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMorseOnchain(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMorseOnchain
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
func skipMorseOnchain(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMorseOnchain
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
					return 0, ErrIntOverflowMorseOnchain
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
					return 0, ErrIntOverflowMorseOnchain
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
				return 0, ErrInvalidLengthMorseOnchain
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMorseOnchain
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMorseOnchain
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMorseOnchain        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMorseOnchain          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMorseOnchain = fmt.Errorf("proto: unexpected end of group")
)
