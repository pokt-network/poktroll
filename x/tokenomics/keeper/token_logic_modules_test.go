package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

// TODO_IMPROVE: Consider using a TestSuite, similar to `x/tokenomics/keeper/keeper_settle_pending_claims_test.go`
// for the TLM based tests in this file.

func TestProcessTokenLogicModules_TLMBurnEqualsMint_Valid(t *testing.T) {
	// Test Parameters
	appInitialStake := math.NewInt(1000000)
	supplierInitialStake := math.NewInt(1000000)
	supplierRevShareRatios := []float32{12.5, 37.5, 50}
	globalComputeUnitsToTokensMultiplier := uint64(1)
	serviceComputeUnitsPerRelay := uint64(1)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By supplier for application in this session

	// Prepare the keepers
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, testkeeper.WithService(*service))
	keepers.SetService(ctx, *service)

	// Ensure the claim is within relay mining bounds
	numTokensClaimed := int64(numRelays * serviceComputeUnitsPerRelay * globalComputeUnitsToTokensMultiplier)
	maxClaimableAmountPerSupplier := appInitialStake.Quo(math.NewInt(sessionkeeper.NumSupplierPerSession))
	require.GreaterOrEqual(t, maxClaimableAmountPerSupplier.Int64(), numTokensClaimed)

	// Retrieve the app and supplier module addresses
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	err := keepers.SharedKeeper.SetParams(ctx, sharedtypes.Params{
		ComputeUnitsToTokensMultiplier: globalComputeUnitsToTokensMultiplier,
	})
	require.NoError(t, err)
	// TODO_TECHDEBT: Setting inflation to zero so we are testing the BurnEqualsMint logic exclusively.
	// Once it is a governance param, update it using the keeper above.
	prevInflationValue := tokenomicskeeper.MintPerClaimedTokenGlobalInflation
	tokenomicskeeper.MintPerClaimedTokenGlobalInflation = 0
	t.Cleanup(func() {
		tokenomicskeeper.MintPerClaimedTokenGlobalInflation = prevInflationValue
	})

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(volatile.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddress(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddress()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, supplierInitialStake)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:    supplierRevShares[0].Address,
		OperatorAddress: supplierRevShares[0].Address,
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: service.Id,
			RevShare:  supplierRevShares,
		}},
	}
	keepers.SetSupplier(ctx, supplier)

	// Query the account and module start balances
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, &claim)
	require.NoError(t, err)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Determine the expected app end stake amount and the expected app burn
	appBurn := math.NewInt(numTokensClaimed)
	expectedAppEndStakeAmount := appInitialStake.Sub(appBurn)

	// Assert that `applicationAddress` staked balance has decreased by the appropriate amount
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	actualAppEndStakeAmount := app.GetStake().Amount
	require.True(t, appIsFound)
	require.Equal(t, expectedAppEndStakeAmount, actualAppEndStakeAmount)

	// Assert that app module balance is *decreased* by the appropriate amount
	// NB: The application module account burns the amount of uPOKT that was held in escrow
	// on behalf of the applications which were serviced in a given session.
	expectedAppModuleEndBalance := appModuleStartBalance.Sub(sdk.NewCoin(volatile.DenomuPOKT, appBurn))
	appModuleEndBalance := getBalance(t, ctx, keepers, appModuleAddress)
	require.NotNil(t, appModuleEndBalance)
	require.EqualValues(t, &expectedAppModuleEndBalance, appModuleEndBalance)

	// Assert that `supplierOperatorAddress` staked balance is *unchanged*
	supplier, supplierIsFound := keepers.GetSupplier(ctx, supplier.GetOperatorAddress())
	require.True(t, supplierIsFound)
	require.Equal(t, &supplierStake, supplier.GetStake())

	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	// NB: Supplier rewards are minted to the supplier module account but then immediately
	// distributed to the supplier accounts which provided service in a given session.
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)

	// Assert that the supplier shareholders account balances have *increased* by
	// the appropriate amount w.r.t token distribution.
	shareAmounts := tokenomicskeeper.GetShareAmountMap(supplierRevShares, appBurn.Uint64())
	for shareHolderAddr, expectedShareAmount := range shareAmounts {
		shareHolderBalance := getBalance(t, ctx, keepers, shareHolderAddr)
		require.Equal(t, int64(expectedShareAmount), shareHolderBalance.Amount.Int64())
	}
}

// DEV_NOTE: Most of the setup here is a copy-paste of TLMBurnEqualsMintValid
// except that the application stake is calculated to explicitly be too low to
// handle all the relays completed.
func TestProcessTokenLogicModules_TLMBurnEqualsMint_Invalid_SupplierExceedsMaxClaimableAmount(t *testing.T) {
	// Test Parameters
	globalComputeUnitsToTokensMultiplier := uint64(1)
	serviceComputeUnitsPerRelay := uint64(1)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By a single supplier for application in this session
	supplierInitialStake := math.NewInt(1000000)
	supplierRevShareRatios := []float32{12.5, 37.5, 50}

	// Prepare the keepers
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, testkeeper.WithService(*service))
	keepers.SetService(ctx, *service)

	// Set up the relays to exceed the max claimable amount
	// Determine the max a supplier can claim
	maxClaimableAmountPerSupplier := int64(numRelays * serviceComputeUnitsPerRelay * globalComputeUnitsToTokensMultiplier)
	// Figure out what the app's initial stake should be to cover the max claimable amount
	appInitialStake := math.NewInt(maxClaimableAmountPerSupplier*sessionkeeper.NumSupplierPerSession + 1)
	// Increase the number of relay such that the supplier did "free work" and would
	// be able to claim more than the max claimable amount.
	numRelays *= 5
	numTokensClaimed := int64(numRelays * serviceComputeUnitsPerRelay * globalComputeUnitsToTokensMultiplier)

	// Retrieve the app and supplier module addresses
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	err := keepers.SharedKeeper.SetParams(ctx, sharedtypes.Params{
		ComputeUnitsToTokensMultiplier: globalComputeUnitsToTokensMultiplier,
	})
	require.NoError(t, err)
	// TODO_TECHDEBT: Setting inflation to zero so we are testing the BurnEqualsMint logic exclusively.
	// Once it is a governance param, update it using the keeper above.
	prevInflationValue := tokenomicskeeper.MintPerClaimedTokenGlobalInflation
	tokenomicskeeper.MintPerClaimedTokenGlobalInflation = 0
	t.Cleanup(func() {
		tokenomicskeeper.MintPerClaimedTokenGlobalInflation = prevInflationValue
	})

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(volatile.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddress(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddress()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, supplierInitialStake)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:    supplierRevShares[0].Address,
		OperatorAddress: supplierRevShares[0].Address,
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: service.Id,
			RevShare:  supplierRevShares,
		}},
	}
	keepers.SetSupplier(ctx, supplier)

	// Query the account and module start balances
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, &claim)
	require.NoError(t, err)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Determine the expected app end stake amount and the expected app burn
	appBurn := math.NewInt(maxClaimableAmountPerSupplier)
	appBurnCoin := sdk.NewCoin(volatile.DenomuPOKT, appBurn)
	expectedAppEndStakeAmount := appInitialStake.Sub(appBurn)

	// Assert that `applicationAddress` staked balance has decreased by the max claimable amount
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	actualAppEndStakeAmount := app.GetStake().Amount
	require.True(t, appIsFound)
	require.Equal(t, expectedAppEndStakeAmount, actualAppEndStakeAmount)

	// Sanity
	require.Less(t, maxClaimableAmountPerSupplier, numTokensClaimed)

	// Assert that app module balance is *decreased* by the appropriate amount
	// NB: The application module account burns the amount of uPOKT that was held in escrow
	// on behalf of the applications which were serviced in a given session.
	expectedAppModuleEndBalance := appModuleStartBalance.Sub(appBurnCoin)
	appModuleEndBalance := getBalance(t, ctx, keepers, appModuleAddress)
	require.NotNil(t, appModuleEndBalance)
	require.EqualValues(t, &expectedAppModuleEndBalance, appModuleEndBalance)

	// Assert that `supplierOperatorAddress` staked balance is *unchanged*
	supplier, supplierIsFound := keepers.GetSupplier(ctx, supplier.GetOperatorAddress())
	require.True(t, supplierIsFound)
	require.Equal(t, &supplierStake, supplier.GetStake())

	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	// NB: Supplier rewards are minted to the supplier module account but then immediately
	// distributed to the supplier accounts which provided service in a given session.
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)

	// Assert that the supplier shareholders account balances have *increased* by
	// the appropriate amount w.r.t token distribution.
	shareAmounts := tokenomicskeeper.GetShareAmountMap(supplierRevShares, appBurn.Uint64())
	for shareHolderAddr, expectedShareAmount := range shareAmounts {
		shareHolderBalance := getBalance(t, ctx, keepers, shareHolderAddr)
		require.Equal(t, int64(expectedShareAmount), shareHolderBalance.Amount.Int64())
	}

	// Check that the expected burn >> effective burn because application is overserviced

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	appOverservicedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventApplicationOverserviced](t,
		events, "poktroll.tokenomics.EventApplicationOverserviced")
	require.Len(t, appOverservicedEvents, 1, "unexpected number of event overserviced events")
	appOverservicedEvent := appOverservicedEvents[0]

	require.Equal(t, app.GetAddress(), appOverservicedEvent.ApplicationAddr)
	require.Equal(t, supplier.GetOperatorAddress(), appOverservicedEvent.SupplierOperatorAddr)
	require.Equal(t, numTokensClaimed, appOverservicedEvent.ExpectedBurn.Amount.Int64())
	require.Equal(t, appBurn, appOverservicedEvent.EffectiveBurn.Amount)
	require.Less(t, appBurn.Int64(), numTokensClaimed)
}

func TestProcessTokenLogicModules_TLMGlobalMint_Valid_MintDistributionCorrect(t *testing.T) {
	// Test Parameters
	appInitialStake := math.NewInt(1000000)
	supplierInitialStake := math.NewInt(1000000)
	supplierRevShareRatios := []float32{12.5, 37.5, 50}
	globalComputeUnitsToTokensMultiplier := uint64(1)
	serviceComputeUnitsPerRelay := uint64(1)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By supplier for application in this session
	numTokensClaimed := float64(numRelays * serviceComputeUnitsPerRelay * globalComputeUnitsToTokensMultiplier)
	validatorConsAddr := sample.ConsAddress()

	// Prepare the keepers
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, testkeeper.WithService(*service), testkeeper.WithProposerAddr(validatorConsAddr))
	keepers.SetService(ctx, *service)

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	err := keepers.SharedKeeper.SetParams(ctx, sharedtypes.Params{
		ComputeUnitsToTokensMultiplier: globalComputeUnitsToTokensMultiplier,
	})
	require.NoError(t, err)

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(volatile.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddress(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddress()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, supplierInitialStake)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:    supplierRevShares[0].Address,
		OperatorAddress: supplierRevShares[0].Address,
		Stake:           &supplierStake,
		Services:        []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id, RevShare: supplierRevShares}},
	}
	keepers.SetSupplier(ctx, supplier)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)

	// Prepare addresses
	daoAddress := authtypes.NewModuleAddress(govtypes.ModuleName)
	appAddress := app.Address
	proposerAddress := sample.AccAddressFromConsAddress(validatorConsAddr)

	// Determine balances before inflation
	daoBalanceBefore := getBalance(t, ctx, keepers, daoAddress.String())
	propBalanceBefore := getBalance(t, ctx, keepers, proposerAddress)
	serviceOwnerBalanceBefore := getBalance(t, ctx, keepers, service.OwnerAddress)
	appBalanceBefore := getBalance(t, ctx, keepers, appAddress)
	supplierShareholderBalancesBefore := make(map[string]*sdk.Coin, len(supplierRevShares))
	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		supplierShareholderBalancesBefore[addr] = getBalance(t, ctx, keepers, addr)
	}

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, &claim)
	require.NoError(t, err)

	// Determine balances after inflation
	daoBalanceAfter := getBalance(t, ctx, keepers, daoAddress.String())
	propBalanceAfter := getBalance(t, ctx, keepers, proposerAddress)
	serviceOwnerBalanceAfter := getBalance(t, ctx, keepers, service.OwnerAddress)
	appBalanceAfter := getBalance(t, ctx, keepers, appAddress)
	supplierShareholderBalancesAfter := make(map[string]*sdk.Coin, len(supplierRevShares))
	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		supplierShareholderBalancesAfter[addr] = getBalance(t, ctx, keepers, addr)
	}

	// Compute mint per actor
	numTokensMinted := numTokensClaimed * tokenomicskeeper.MintPerClaimedTokenGlobalInflation
	daoMint := math.NewInt(int64(numTokensMinted * tokenomicskeeper.MintAllocationDAO))
	propMint := math.NewInt(int64(numTokensMinted * tokenomicskeeper.MintAllocationProposer))
	serviceOwnerMint := math.NewInt(int64(numTokensMinted * tokenomicskeeper.MintAllocationSourceOwner))
	appMint := math.NewInt(int64(numTokensMinted * tokenomicskeeper.MintAllocationApplication))
	supplierMint := float32(numTokensMinted * tokenomicskeeper.MintAllocationSupplier)

	// Ensure the balance was increase be the appropriate amount
	require.Equal(t, daoBalanceBefore.Amount.Add(daoMint), daoBalanceAfter.Amount)
	require.Equal(t, propBalanceBefore.Amount.Add(propMint), propBalanceAfter.Amount)
	require.Equal(t, serviceOwnerBalanceBefore.Amount.Add(serviceOwnerMint), serviceOwnerBalanceAfter.Amount)
	require.Equal(t, appBalanceBefore.Amount.Add(appMint), appBalanceAfter.Amount)
	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		balanceBefore := supplierShareholderBalancesBefore[addr]
		balanceAfter := supplierShareholderBalancesAfter[addr].Amount.Int64()
		mintShare := int64(supplierMint * revShare.RevSharePercentage / 100)
		rewardShare := int64(float32(numTokensClaimed) * revShare.RevSharePercentage / 100)
		balanceIncrease := math.NewInt(mintShare + rewardShare)
		expectedBalanceAfter := balanceBefore.Amount.Add(balanceIncrease).Int64()
		// TODO_MAINNET: Remove the InDelta check and use the exact amount once the floating point arithmetic is fixed
		acceptableRoundingDelta := tokenomicskeeper.MintDistributionAllowableTolerancePercent * float64(balanceAfter)
		require.InDelta(t, expectedBalanceAfter, balanceAfter, acceptableRoundingDelta)
	}
}

func TestProcessTokenLogicModules_AppNotFound(t *testing.T) {
	keeper, ctx, _, supplierOperatorAddr, service := testkeeper.TokenomicsKeeperWithActorAddrs(t)

	// The base claim whose root will be customized for testing purposes
	numRelays := uint64(42)
	numComputeUnits := numRelays * service.ComputeUnitsPerRelay
	claim := prooftypes.Claim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      sample.AccAddress(), // Random address
			ServiceId:               service.Id,
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSumAndCount(numComputeUnits, numRelays),
	}

	// Process the token logic modules
	err := keeper.ProcessTokenLogicModules(ctx, &claim)
	require.Error(t, err)
	require.ErrorIs(t, err, tokenomicstypes.ErrTokenomicsApplicationNotFound)
}

func TestProcessTokenLogicModules_ServiceNotFound(t *testing.T) {
	keeper, ctx, appAddr, supplierOperatorAddr, service := testkeeper.TokenomicsKeeperWithActorAddrs(t)

	numRelays := uint64(42)
	numComputeUnits := numRelays * service.ComputeUnitsPerRelay
	claim := prooftypes.Claim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			ServiceId:               "non_existent_svc",
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSumAndCount(numComputeUnits, numRelays),
	}

	// Execute test function
	err := keeper.ProcessTokenLogicModules(ctx, &claim)

	require.Error(t, err)
	require.ErrorIs(t, err, tokenomicstypes.ErrTokenomicsServiceNotFound)
}

func TestProcessTokenLogicModules_InvalidRoot(t *testing.T) {
	keeper, ctx, appAddr, supplierOperatorAddr, service := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	numRelays := uint64(42)

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
			errExpected: true,
		},
		{
			desc: "correct size but invalid value",
			root: func() []byte {
				root := bytes.Repeat([]byte("a"), protocol.TrieRootSize)
				// Change the last byte to 'b'
				root = append(root[:len(root)-1], 'b')
				return root
			}(),
			errExpected: true,
		},
		{
			desc: "correct size and a valid value",
			root: func() []byte {
				root := testproof.SmstRootWithSumAndCount(numRelays, numRelays)
				return root[:]
			}(),
			errExpected: false,
		},
	}

	// Iterate over each test case
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Setup claim by copying the testproof.BaseClaim and updating the root
			claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, 0)
			claim.RootHash = smt.MerkleRoot(test.root[:])

			// Execute test function
			err := keeper.ProcessTokenLogicModules(ctx, &claim)

			// Assert the error
			if test.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProcessTokenLogicModules_InvalidClaim(t *testing.T) {
	keeper, ctx, appAddr, supplierOperatorAddr, service := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	numRelays := uint64(42)

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
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
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
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader = nil
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderNil,
		},
		{
			desc: "Claim with invalid session id",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader.SessionId = ""
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid application address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader.ApplicationAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSessionHeaderInvalid,
		},
		{
			desc: "Claim with invalid supplier operator address",
			claim: func() *prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SupplierOperatorAddress = "invalid address"
				return &claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSupplierOperatorAddressInvalid,
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
				return keeper.ProcessTokenLogicModules(ctx, test.claim)
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

// prepareTestClaim uses the given number of relays and compute unit per relay in the
// service provided to set up the test claim correctly.
func prepareTestClaim(
	numRelays uint64,
	service *sharedtypes.Service,
	app *apptypes.Application,
	supplier *sharedtypes.Supplier,
) prooftypes.Claim {
	numComputeUnits := numRelays * service.ComputeUnitsPerRelay
	return prooftypes.Claim{
		SupplierOperatorAddress: supplier.OperatorAddress,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      app.Address,
			ServiceId:               service.Id,
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSumAndCount(numComputeUnits, numRelays),
	}
}

// prepareTestService creates a service with the given compute units per relay.
func prepareTestService(serviceComputeUnitsPerRelay uint64) *sharedtypes.Service {
	return &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: serviceComputeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
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
