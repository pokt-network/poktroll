package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestSettleSessionAccounting_HandleAppGoingIntoDebt(t *testing.T) {
	t.Skip("TODO_MAINNET: Add coverage of the design choice made for how to handle this scenario.")

	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil)

	// Create a service that can be registered in the application and used in the claims
	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}

	// Add a new application
	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			&sharedtypes.ApplicationServiceConfig{
				Service: service,
			},
		},
	}
	keepers.SetApplication(ctx, app)

	// Add a new supplier
	supplierStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &supplierStake,
	}
	keepers.SetSupplier(ctx, supplier)

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplier.Address,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: app.Address,
			Service: &sharedtypes.Service{
				Id: service.Id,
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSum(appStake.Amount.Uint64() + 1), // More than the app stake
	}

	err := keepers.SettleSessionAccounting(ctx, &claim)
	require.NoError(t, err)
}

func TestSettleSessionAccounting_ValidAccounting(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil)
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to 1 to simplify expectation calculations.
	err := keepers.Keeper.SetParams(ctx, tokenomicstypes.Params{
		ComputeUnitsToTokensMultiplier: 1,
	})
	require.NoError(t, err)

	// Create a service that can be registered in the application and used in the claims
	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}
	// Add a new application
	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	// NB: Ensure a non-zero app stake end balance to assert against.
	expectedAppEndStakeAmount := cosmostypes.NewCoin("upokt", math.NewInt(420))
	expectedAppBurn := appStake.Sub(expectedAppEndStakeAmount)
	app := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			&sharedtypes.ApplicationServiceConfig{
				Service: service,
			},
		},
	}
	keepers.SetApplication(ctx, app)

	// Query application balance prior to the accounting.
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())

	// Query application module balance prior to the accounting.
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &supplierStake,
	}
	keepers.SetSupplier(ctx, supplier)

	// Query supplier balance prior to the accounting.
	supplierStartBalance := getBalance(t, ctx, keepers, supplier.GetAddress())

	// Query supplier module balance prior to the accounting.
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplier.Address,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: app.Address,
			Service: &sharedtypes.Service{
				Id: service.Id,
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSum(expectedAppBurn.Amount.Uint64()),
	}

	err = keepers.SettleSessionAccounting(ctx, &claim)
	require.NoError(t, err)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Assert that `applicationAddress` staked balance has decreased by the appropriate amount
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	require.True(t, appIsFound)
	require.Equal(t, &expectedAppEndStakeAmount, app.GetStake())

	// Assert that `apptypes.ModuleName` account module balance is *decreased* by the appropriate amount
	// NB: The application module account burns the amount of uPOKT that was held in escrow
	// on behalf of the applications which were serviced in a given session.
	appModuleEndBalance := getBalance(t, ctx, keepers, appModuleAddress)
	expectedAppModuleEndBalance := appModuleStartBalance.Sub(expectedAppBurn)
	require.NotNil(t, appModuleEndBalance)
	require.EqualValues(t, &expectedAppModuleEndBalance, appModuleEndBalance)

	// Assert that `supplierAddress` account balance has *increased* by the appropriate amount
	supplierEndBalance := getBalance(t, ctx, keepers, supplier.GetAddress())
	expectedSupplierBalance := supplierStartBalance.Add(expectedAppBurn)
	require.EqualValues(t, &expectedSupplierBalance, supplierEndBalance)

	// Assert that `supplierAddress` staked balance is *unchanged*
	supplier, supplierIsFound := keepers.GetSupplier(ctx, supplier.GetAddress())
	require.True(t, supplierIsFound)
	require.Equal(t, &supplierStake, supplier.GetStake())

	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	// NB: Supplier rewards are minted to the supplier module account but then immediately
	// distributed to the supplier accounts which provided service in a given session.
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)
}

func TestSettleSessionAccounting_AppStakeTooLow(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil)
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to 1 to simplify expectation calculations.
	err := keepers.Keeper.SetParams(ctx, tokenomicstypes.Params{
		ComputeUnitsToTokensMultiplier: 1,
	})
	require.NoError(t, err)

	// Create a service that can be registered in the application and used in the claims
	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}
	// Add a new application
	appStake := cosmostypes.NewCoin("upokt", math.NewInt(40000))
	expectedAppEndStakeZeroAmount := cosmostypes.NewCoin("upokt", math.NewInt(0))
	expectedAppBurn := appStake.AddAmount(math.NewInt(2000))
	app := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			&sharedtypes.ApplicationServiceConfig{
				Service: service,
			},
		},
	}
	keepers.SetApplication(ctx, app)

	// Query application balance prior to the accounting.
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())

	// Query application module balance prior to the accounting.
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &supplierStake,
	}
	keepers.SetSupplier(ctx, supplier)

	// Query supplier balance prior to the accounting.
	supplierStartBalance := getBalance(t, ctx, keepers, supplier.GetAddress())

	// Query supplier module balance prior to the accounting.
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplier.Address,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: app.Address,
			Service: &sharedtypes.Service{
				Id: service.Id,
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSum(expectedAppBurn.Amount.Uint64()),
	}

	err = keepers.SettleSessionAccounting(ctx, &claim)
	require.NoError(t, err)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Assert that `applicationAddress` staked balance has gone to zero
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	require.True(t, appIsFound)
	require.Equal(t, &expectedAppEndStakeZeroAmount, app.GetStake())

	// Assert that `apptypes.ModuleName` account module balance is *decreased* by the appropriate amount
	appModuleEndBalance := getBalance(t, ctx, keepers, appModuleAddress)
	expectedAppModuleEndBalance := appModuleStartBalance.Sub(appStake)
	require.NotNil(t, appModuleEndBalance)
	require.EqualValues(t, &expectedAppModuleEndBalance, appModuleEndBalance)

	// Assert that `supplierAddress` account balance has *increased* by the appropriate amount
	supplierEndBalance := getBalance(t, ctx, keepers, supplier.GetAddress())
	require.NotNil(t, supplierEndBalance)

	expectedSupplierBalance := supplierStartBalance.Add(expectedAppBurn)
	require.EqualValues(t, &expectedSupplierBalance, supplierEndBalance)

	// Assert that `supplierAddress` staked balance is *unchanged*
	supplier, supplierIsFound := keepers.GetSupplier(ctx, supplier.GetAddress())
	require.True(t, supplierIsFound)
	require.Equal(t, &supplierStake, supplier.GetStake())

	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()

	appAddrAttribute, _ := events.GetAttributes("application_addr")
	expectedBurnAttribute, _ := events.GetAttributes("expected_burn")
	effectiveBurnAttribute, _ := events.GetAttributes("effective_burn")

	require.Equal(t, 1, len(appAddrAttribute))
	require.Equal(t, fmt.Sprintf("\"%s\"", app.GetAddress()), appAddrAttribute[0].Value)

	var expectedBurnEventCoin, effectiveBurnEventCoin cosmostypes.Coin
	err = json.Unmarshal([]byte(expectedBurnAttribute[0].Value), &expectedBurnEventCoin)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(effectiveBurnAttribute[0].Value), &effectiveBurnEventCoin)
	require.NoError(t, err)

	require.EqualValues(t, expectedAppBurn, expectedBurnEventCoin)
	require.Greater(t, expectedBurnEventCoin.Amount.Uint64(), effectiveBurnEventCoin.Amount.Uint64())
}

func TestSettleSessionAccounting_AppNotFound(t *testing.T) {

	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}

	keeper, ctx, _, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t, service)

	// The base claim whose root will be customized for testing purposes
	claim := prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: sample.AccAddress(), // Random address
			Service: &sharedtypes.Service{
				Id: service.Id,
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

	// Create a service that can be registered in the application and used in the claims
	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}

	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t, service)

	// Define test cases
	tests := []struct {
		desc        string
		root        []byte // smst.MerkleSumRoot
		errExpected bool
	}{
		{
			desc:        "Nil Root",
			root:        nil,
			errExpected: true,
		},
		{
			desc:        fmt.Sprintf("Less than %d bytes", protocol.TrieRootSize),
			root:        make([]byte, protocol.TrieRootSize-1), // Less than expected number of bytes
			errExpected: true,
		},
		{
			desc:        fmt.Sprintf("More than %d bytes", protocol.TrieRootSize),
			root:        make([]byte, protocol.TrieRootSize+1), // More than expected number of bytes
			errExpected: true,
		},
		{
			desc: "correct size but empty",
			root: func() []byte {
				root := make([]byte, protocol.TrieRootSize) // All 0s
				return root[:]
			}(),
			errExpected: false,
		},
		{
			desc: "correct size but invalid value",
			root: func() []byte {
				return bytes.Repeat([]byte("a"), protocol.TrieRootSize)
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
			// Setup claim by copying the testproof.BaseClaim and updating the root
			claim := testproof.BaseClaim(appAddr, supplierAddr, 0, service.Id)
			claim.RootHash = smt.MerkleRoot(test.root[:])

			// Execute test function
			err := keeper.SettleSessionAccounting(ctx, &claim)

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
	// Create a service that can be registered in the application and used in the claims
	service := &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}

	keeper, ctx, appAddr, supplierAddr := testkeeper.TokenomicsKeeperWithActorAddrs(t, service)

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
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42, service.Id)
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
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42, service.Id)
				claim.SessionHeader = nil
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderNil,
		},
		{
			desc: "Claim with invalid session id",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42, service.Id)
				claim.SessionHeader.SessionId = ""
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid application address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42, service.Id)
				claim.SessionHeader.ApplicationAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid supplier address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(appAddr, supplierAddr, 42, service.Id)
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

func getBalance(
	t *testing.T,
	ctx context.Context,
	bankKeeper tokenomicstypes.BankKeeper,
	accountAddr string,
) *cosmostypes.Coin {

	appBalanceRes, err := bankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: accountAddr,
		Denom:   "upokt",
	})
	require.NoError(t, err)

	balance := appBalanceRes.GetBalance()
	require.NotNil(t, balance)

	return balance
}
