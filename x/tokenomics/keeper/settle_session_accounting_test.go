package keeper_test

import (
	"bytes"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestSettleSessionAccounting_HandleAppGoingIntoDebt(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil)

	// Add a new application
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
	}
	keepers.SetApplication(ctx, app)

	// Add a new supplier
	supplierStake := types.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &supplierStake,
	}

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplier.Address,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: app.Address,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svcName1",
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSum(appStake.Amount.Uint64() + 1), // More than the app stake
	}

	err := keepers.SettleSessionAccounting(ctx, &claim)
	require.NoError(t, err)
	// TODO_TEST: Need to make sure the application is unstaked at this point in time.
}

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
	keeper, ctx, _, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t)

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
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSum(42),
	}

	err := keeper.SettleSessionAccounting(ctx, &claim)
	require.Error(t, err)
	require.ErrorIs(t, err, tokenomicstypes.ErrTokenomicsApplicationNotFound)
}

func TestSettleSessionAccounting_InvalidRoot(t *testing.T) {
	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t)

	rootHashSizeBytes := smt.SmstRootSizeBytes
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
			desc:        fmt.Sprintf("Less than %d bytes", rootHashSizeBytes),
			root:        make([]byte, rootHashSizeBytes-1), // Less than expected number of bytes
			errExpected: true,
		},
		{
			desc:        fmt.Sprintf("More than %d bytes", rootHashSizeBytes),
			root:        make([]byte, rootHashSizeBytes+1), // More than expected number of bytes
			errExpected: true,
		},
		{
			desc: "correct size but empty",
			root: func() []byte {
				root := make([]byte, rootHashSizeBytes) // All 0s
				return root[:]
			}(),
			errExpected: false,
		},
		{
			desc: "correct size but invalid value",
			root: func() []byte {
				return bytes.Repeat([]byte("a"), rootHashSizeBytes)
			}(),
			errExpected: true,
		},
		{
			desc: "correct size and a valid value",
			root: func() []byte {
				root := testproof.SmstRootWithSum(42)
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

			// Setup claim by copying the testproof.BaseClaim and updating the root
			claim := testproof.BaseClaim(appAddr, supplierAddr, 0)
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
	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t)

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
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42)
				return &claim
			}(),
			errExpected: false,
		},
		{
			desc:        "Nil Claim",
			claim:       nil,
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsClaimNil,
		},
		{
			desc: "Claim with nil session header",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader = nil
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderNil,
		},
		{
			desc: "Claim with invalid session id",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader.SessionId = ""
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid application address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42)
				claim.SessionHeader.ApplicationAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid supplier address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42)
				claim.SupplierAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSupplierAddressInvalid,
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
