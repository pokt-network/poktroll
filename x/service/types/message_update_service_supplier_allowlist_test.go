package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateServiceSupplierAllowlist_ValidateBasic(t *testing.T) {
	owner := sample.AccAddressBech32()
	addr := sample.AccAddressBech32()
	otherAddr := sample.AccAddressBech32()

	sampleAddrs := func(n int) []string {
		addrs := make([]string, n)
		for i := range addrs {
			addrs[i] = sample.AccAddressBech32()
		}
		return addrs
	}

	tests := []struct {
		desc        string
		msg         MsgUpdateServiceSupplierAllowlist
		expectedErr error
	}{
		{
			desc: "valid add",
			msg:  MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: []string{addr}},
		},
		{
			desc: "valid remove only",
			msg:  MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Remove: []string{addr}},
		},
		{
			desc:        "non-canonical owner",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: strings.ToUpper(owner), ServiceId: "svc1", Add: []string{addr}},
			expectedErr: ErrServiceInvalidAddress,
		},
		{
			desc:        "non-canonical add entry",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: []string{strings.ToUpper(addr)}},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc:        "invalid remove entry",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Remove: []string{"pokt1invalid"}},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc:        "duplicate within add",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: []string{addr, addr}},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc:        "duplicate within remove",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Remove: []string{addr, addr}},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc:        "same address in add and remove",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: []string{addr, otherAddr}, Remove: []string{addr}},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc:        "101 entries across add and remove",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: sampleAddrs(60), Remove: sampleAddrs(41)},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
		{
			desc: "exactly 100 entries across add and remove",
			msg:  MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1", Add: sampleAddrs(60), Remove: sampleAddrs(40)},
		},
		{
			desc:        "empty service ID",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, Add: []string{addr}},
			expectedErr: ErrServiceMissingID,
		},
		{
			desc:        "no add and no remove",
			msg:         MsgUpdateServiceSupplierAllowlist{OwnerAddress: owner, ServiceId: "svc1"},
			expectedErr: ErrServiceInvalidSupplierAllowlist,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
