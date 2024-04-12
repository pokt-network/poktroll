package keeper_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_TEST(@bryanchriswhite, @Olshansk): Improve tokenomics tests (i.e. checking balances)
// once in-memory network integration tests are supported.

func TestSettleSessionAccounting_ValidAccounting(t *testing.T) {
	t.Skip("TODO_BLOCKER(@Olshansk): Add E2E and integration tests so we validate the actual state changes of the bank & account keepers.")
	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	// Assert that `supplierAddress` account balance has *increased* by the appropriate amount
	// Assert that `supplierAddress` staked balance is *unchanged*
	// Assert that `apptypes.ModuleName` account module balance is *unchanged*
	// Assert that `applicationAddress` account balance is *unchanged*
	// Assert that `applicationAddress` staked balance has decreased by the appropriate amount
}

func TestSettleSessionAccounting_AppStakeTooLow(t *testing.T) {
	t.Skip("TODO_BLOCKER(@Olshansk): Add E2E and integration tests so we validate the actual state changes of the bank & account keepers.")
	// Assert that `suppliertypes.Address` account balance has *increased* by the appropriate amount
	// Assert that `applicationAddress` account staked balance has gone to zero
	// Assert on whatever logic we have for slashing the application or other
}

func TestSettleSessionAccounting_AppNotFound(t *testing.T) {
	keeper, ctx, _, supplierAddr := testkeeper.TokenomicsKeeper(t)

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: sample.AccAddress(), // Random address
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svcName1",
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   sessionkeeper.GetSessionEndBlockHeight(1),
		},
		RootHash: smstRootWithSum(42),
	}

	err := keeper.SettleSessionAccounting(ctx, &claim)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrTokenomicsApplicationNotFound)
}

func TestSettleSessionAccounting_InvalidRoot(t *testing.T) {
	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeper(t)

	// Define test cases
	tests := []struct {
		desc        string
		root        []byte // smst.MerkleRoot
		errExpected bool
	}{
		{
			desc:        "Nil Root",
			root:        nil,
			errExpected: true,
		},
		{
			desc:        "Less than 40 bytes",
			root:        make([]byte, 39), // Less than 40 bytes
			errExpected: true,
		},
		{
			desc:        "More than 40 bytes",
			root:        make([]byte, 41), // More than 40 bytes
			errExpected: true,
		},
		{
			desc: "40 bytes but empty",
			root: func() []byte {
				root := make([]byte, 40) // 40-byte slice of all 0s
				return root[:]
			}(),
			errExpected: false,
		},
		{
			desc: "40 bytes but has an invalid value",
			root: func() []byte {
				var root [40]byte
				copy(root[:], []byte("This text is exactly 40 characters!!!!!!"))
				return root[:]
			}(),
			errExpected: true,
		},
		{
			desc: "40 bytes and has a valid value",
			root: func() []byte {
				root := smstRootWithSum(42)
				return root[:]
			}(),
			errExpected: false,
		},
	}

	// Iterate over each test case
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Use defer-recover to catch any panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Test panicked: %s", r)
				}
			}()

			// Setup claim by copying the baseClaim and updating the root
			claim := baseClaim(appAddr, supplierAddr, 0)
			claim.RootHash = smt.MerkleRoot(test.root[:])

			// Execute test function
			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic occurred: %v", r)
					}
				}()
				return keeper.SettleSessionAccounting(ctx, &claim)
			}()

			// Assert the error
			if test.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSettleSessionAccounting_InvalidClaim(t *testing.T) {
	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeper(t)

	// Define test cases
	tests := []struct {
		desc        string
		claim       *prooftypes.Claim
		errExpected bool
		expectErr   error
	}{

		{
			desc: "Valid Claim",
			claim: func() *prooftypes.Claim {
				claim := baseClaim(appAddr, supplierAddr, 42)
				return &claim
			}(),
			errExpected: false,
		},
		{
			desc:        "Nil Claim",
			claim:       nil,
			errExpected: true,
			expectErr:   types.ErrTokenomicsClaimNil,
		},
		{
			desc: "Claim with nil session header",
			claim: func() *prooftypes.Claim {
				claim := baseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader = nil
				return &claim
			}(),
			errExpected: true,
			expectErr:   types.ErrTokenomicsSessionHeaderNil,
		},
		{
			desc: "Claim with invalid session id",
			claim: func() *prooftypes.Claim {
				claim := baseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader.SessionId = ""
				return &claim
			}(),
			errExpected: true,
			expectErr:   types.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid application address",
			claim: func() *prooftypes.Claim {
				claim := baseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader.ApplicationAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   types.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid supplier address",
			claim: func() *prooftypes.Claim {
				claim := baseClaim(appAddr, supplierAddr, 42)
				claim.SupplierAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   types.ErrTokenomicsSupplierAddressInvalid,
		},
	}

	// Iterate over each test case
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
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
				return keeper.SettleSessionAccounting(ctx, test.claim)
			}()

			// Assert the error
			if test.errExpected {
				require.Error(t, err)
				require.ErrorIs(t, err, test.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func baseClaim(appAddr, supplierAddr string, sum uint64) prooftypes.Claim {
	return prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: appAddr,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svcName1",
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   sessionkeeper.GetSessionEndBlockHeight(1),
		},
		RootHash: smstRootWithSum(sum),
	}
}

func smstRootWithSum(sum uint64) smt.MerkleRoot {
	root := make([]byte, 40)
	copy(root[:32], []byte("This is exactly 32 characters!!!"))
	binary.BigEndian.PutUint64(root[32:], sum)
	return smt.MerkleRoot(root)
}
