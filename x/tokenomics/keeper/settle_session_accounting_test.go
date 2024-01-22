package keeper_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_BLOCKER, TODO_ADDTEST(@Olshansk): Add E2E and integration tests for
// the actual address values when the bank and account keeper is not mocked.

func TestSettleSessionAccounting_InvalidRoot(t *testing.T) {
	keeper, ctx := testkeeper.TokenomicsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Define test cases
	testCases := []struct {
		desc      string
		root      []byte // smst.MerkleRoot
		expectErr bool
	}{
		{
			desc:      "Nil Root",
			root:      nil,
			expectErr: true,
		},
		{
			desc:      "Less than 40 bytes",
			root:      []byte("less than 40 bytes"),
			expectErr: true,
		},
		{
			desc: "More than 40 bytes",
			root: []byte("more than 40 bytes, this string is too long"),
			// TODO_IN_THIS_PR: This should be true, but we need to fix it on the SMT side
			expectErr: false,
		},
		{
			desc: "40 bytes but empty",
			root: func() []byte {
				root := [40]byte{}
				return root[:]
			}(),
			// TODO_IN_THIS_PR: This should be true, but we need to fix it on the SMT side
			expectErr: false,
		},
		{
			desc: "40 bytes but has an invalid value",
			root: func() []byte {
				var root [40]byte
				copy(root[:], []byte("exact 40 byte string............."))
				return root[:]
			}(),
			expectErr: true,
		},
		{
			desc: "40 bytes and has a valid value",
			root: func() []byte {
				var root [40]byte
				// Put unsigned value of 100 into the first 8 bytes
				binary.BigEndian.PutUint64(root[:8], 100)
				// Copy additional bytes if needed
				copy(root[8:], []byte("exact 40 byte string..."))
				return root[:]
			}(),
			expectErr: false,
		},
	}

	// Iterate over each test case
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Use defer-recover to catch any panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Test panicked: %s", r)
				}
			}()

			// Setup claim
			claim := suppliertypes.Claim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svc1",
					},
					SessionStartBlockHeight: 1,
					SessionId:               "1",
					SessionEndBlockHeight:   5,
				},
				RootHash: smt.MerkleRoot(tc.root[:]),
			}

			// Execute test function
			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic occurred: %v", r)
					}
				}()
				return keeper.SettleSessionAccounting(wctx, &claim)
			}()

			// Assert the error
			if tc.expectErr {
				require.Error(t, err, "Test case: %s", tc.desc)
			} else {
				require.NoError(t, err, "Test case: %s", tc.desc)
			}
		})
	}
}

func TestSettleSessionAccounting_InvalidClaim(t *testing.T) {
	keeper, ctx := testkeeper.TokenomicsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	var root [40]byte
	binary.BigEndian.PutUint64(root[:8], 100)
	copy(root[8:], []byte("exact 40 byte string..."))
	merkleRoot := smt.MerkleRoot(root[:])

	claim := suppliertypes.Claim{
		SupplierAddress: sample.AccAddress(),
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: sample.AccAddress(),
			Service: &sharedtypes.Service{
				Id: "svc1",
			},
			SessionStartBlockHeight: 1,
			SessionId:               "1",
			SessionEndBlockHeight:   5,
		},
		RootHash: merkleRoot,
	}

	// Define test cases
	testCases := []struct {
		desc      string
		claim     *suppliertypes.Claim
		expectErr bool
	}{
		{
			desc:      "Nil Claim",
			claim:     nil,
			expectErr: true,
		},
		{
			desc: "Claim with nil root",
			claim: func() *suppliertypes.Claim {
				c := claim
				c.RootHash = nil
				return &c
			}(),
			expectErr: true,
		},
		{
			desc:      "Valid Claim",
			claim:     &claim,
			expectErr: false,
		},
	}

	// Iterate over each test case
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Use defer-recover to catch any panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Test panicked: %s", r)
				}
			}()

			// Execute test function
			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic occurred: %v", r)
					}
				}()
				return keeper.SettleSessionAccounting(wctx, tc.claim)
			}()

			// Assert the error
			if tc.expectErr {
				require.Error(t, err, "Test case: %s", tc.desc)
			} else {
				require.NoError(t, err, "Test case: %s", tc.desc)
			}
		})
	}
}
