package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	cosmoslog "cosmossdk.io/log"
	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/encoding"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

// TODO_IMPROVE: Consider using a TestSuite, similar to `x/tokenomics/keeper/keeper_settle_pending_claims_test.go`
// for the TLM based tests in this file.

func TestProcessTokenLogicModules_TLMBurnEqualsMint_AppToSupplierOnly_Valid(t *testing.T) {
	// Test Parameters
	appInitialStake := apptypes.DefaultMinStake.Amount.Mul(cosmosmath.NewInt(2))
	supplierInitialStake := cosmosmath.NewInt(1000000)
	supplierRevShareRatios := []uint64{12, 38, 50}
	// Set the cost denomination of a single compute unit to pPOKT (i.e. 1/compute_unit_cost_granularity)
	globalComputeUnitCostGranularity := uint64(1000000)
	globalComputeUnitsToTokensMultiplier := uint64(1) * globalComputeUnitCostGranularity
	serviceComputeUnitsPerRelay := uint64(1)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By supplier for application in this session

	// Prepare the keepers with only relay burn equals mint TLM
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t,
		cosmoslog.NewNopLogger(),
		testkeeper.WithService(*service),
		testkeeper.WithDefaultModuleBalances(),
		testkeeper.WithTokenLogicModules([]tlm.TokenLogicModule{
			tlm.NewRelayBurnEqualsMintTLM(),
		}),
	)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(ctx, *service)

	// Ensure the claim is within relay mining bounds
	numSuppliersPerSession := int64(keepers.SessionKeeper.GetParams(ctx).NumSuppliersPerSession)
	numTokensClaimed := getNumTokensClaimed(
		numRelays,
		serviceComputeUnitsPerRelay,
		globalComputeUnitsToTokensMultiplier,
		globalComputeUnitCostGranularity,
	)
	maxClaimableAmountPerSupplier := appInitialStake.Quo(cosmosmath.NewInt(numSuppliersPerSession))
	require.GreaterOrEqual(t, maxClaimableAmountPerSupplier.Int64(), numTokensClaimed)

	// Retrieve the app and supplier module addresses
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	sharedParams.ComputeUnitsToTokensMultiplier = globalComputeUnitsToTokensMultiplier
	err := keepers.SharedKeeper.SetParams(ctx, sharedParams)
	require.NoError(t, err)

	// Setting inflation to zero so we are testing the BurnEqualsMint logic exclusively.
	tokenomicsParams := keepers.Keeper.GetParams(ctx)
	tokenomicsParams.GlobalInflationPerClaim = 0
	tokenomicsParams.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Dao:         0,
		Proposer:    0,
		Supplier:    1,
		SourceOwner: 0,
		Application: 0,
	}
	err = keepers.Keeper.SetParams(ctx, tokenomicsParams)
	require.NoError(t, err)

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(pocket.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddressBech32()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}
	services := []*sharedtypes.SupplierServiceConfig{{
		ServiceId: service.Id,
		RevShare:  supplierRevShares,
	}}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierInitialStake)
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplierRevShares[0].Address,
		services, 1, 0,
	)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:         supplierRevShares[0].Address,
		OperatorAddress:      supplierRevShares[0].Address,
		Stake:                &supplierStake,
		Services:             services,
		ServiceConfigHistory: serviceConfigHistory,
	}
	keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	// Query the account and module start balances
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)
	pendingResult := tlm.NewClaimSettlementResult(claim)

	settlementContext := tokenomicskeeper.NewSettlementContext(
		ctx,
		keepers.Keeper,
		keepers.Logger(),
	)

	err = settlementContext.ClaimCacheWarmUp(ctx, &claim)
	require.NoError(t, err)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
	require.NoError(t, err)

	// Execute the pending results
	pendingResults := make(tlm.ClaimSettlementResults, 0)
	pendingResults.Append(pendingResult)
	err = keepers.ExecutePendingSettledResults(cosmostypes.UnwrapSDKContext(ctx), pendingResults)
	require.NoError(t, err)

	// Persist the actors state
	settlementContext.FlushAllActorsToStore(ctx)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Determine the expected app end stake amount and the expected app burn
	appBurn := cosmosmath.NewInt(numTokensClaimed)
	expectedAppEndStakeAmount := appInitialStake.Sub(appBurn)

	// Assert that `applicationAddress` staked balance has decreased by the appropriate amount
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	actualAppEndStakeAmount := app.GetStake().Amount
	require.True(t, appIsFound)
	require.Equal(t, expectedAppEndStakeAmount, actualAppEndStakeAmount)

	// Assert that app module balance is *decreased* by the appropriate amount
	// DEV_NOTE: The application module account burns the amount of uPOKT that was held in escrow
	// on behalf of the applications which were serviced in a given session.
	expectedAppModuleEndBalance := appModuleStartBalance.Sub(cosmostypes.NewCoin(pocket.DenomuPOKT, appBurn))
	appModuleEndBalance := getBalance(t, ctx, keepers, appModuleAddress)
	require.NotNil(t, appModuleEndBalance)
	require.EqualValues(t, &expectedAppModuleEndBalance, appModuleEndBalance)

	// Assert that `supplierOperatorAddress` staked balance is *unchanged*
	supplier, supplierIsFound := keepers.GetSupplier(ctx, supplier.GetOperatorAddress())
	require.True(t, supplierIsFound)
	require.Equal(t, &supplierStake, supplier.GetStake())

	// Assert that `suppliertypes.ModuleName` account module balance is *unchanged*
	// DEV_NOTE: Supplier rewards are minted to the supplier module account but then immediately
	// distributed to the supplier accounts which provided service in a given session.
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)

	// Assert that the supplier shareholders account balances have *increased* by
	// the appropriate amount w.r.t token distribution.
	// The supplier gets a percentage of the total settlement based on MintEqualsBurnClaimDistribution
	supplierAllocation := appBurn.MulRaw(int64(keepers.Keeper.GetParams(ctx).MintEqualsBurnClaimDistribution.Supplier * 100)).QuoRaw(100)
	shareAmounts := tlm.GetShareAmountMap(supplierRevShares, supplierAllocation)
	for shareHolderAddr, expectedShareAmount := range shareAmounts {
		shareHolderBalance := getBalance(t, ctx, keepers, shareHolderAddr)
		require.Equal(t, expectedShareAmount, shareHolderBalance.Amount)
	}
}

// DEV_NOTE: Most of the setup here is a copy-paste of TLMBurnEqualsMintValid
// except that the application stake is calculated to explicitly be too low to
// handle all the relays completed.
func TestProcessTokenLogicModules_TLMBurnEqualsMint_AppToSupplierExceedsMaxClaimableAmount_Valid(t *testing.T) {
	// Test Parameters
	// Set the cost denomination of a single compute unit to pPOKT (i.e. 1/compute_unit_cost_granularity)
	globalComputeUnitCostGranularity := uint64(1000000)
	globalComputeUnitsToTokensMultiplier := uint64(1) * globalComputeUnitCostGranularity
	serviceComputeUnitsPerRelay := uint64(100)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By a single supplier for application in this session
	supplierInitialStake := cosmosmath.NewInt(1000000)
	supplierRevShareRatios := []uint64{12, 38, 50}

	// Prepare the keepers with only relay burn equals mint TLM
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t,
		cosmoslog.NewNopLogger(),
		testkeeper.WithService(*service),
		testkeeper.WithDefaultModuleBalances(),
		testkeeper.WithTokenLogicModules([]tlm.TokenLogicModule{
			tlm.NewRelayBurnEqualsMintTLM(),
		}),
	)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(ctx, *service)

	// Set up the relays to exceed the max claimable amount
	// Determine the max a supplier can claim
	maxClaimableAmountPerSupplier := getNumTokensClaimed(
		numRelays,
		serviceComputeUnitsPerRelay,
		globalComputeUnitsToTokensMultiplier,
		globalComputeUnitCostGranularity,
	)
	// Figure out what the app's initial stake should be to cover the max claimable amount
	numSuppliersPerSession := int64(keepers.SessionKeeper.GetParams(ctx).NumSuppliersPerSession)
	appInitialStake := cosmosmath.NewInt(maxClaimableAmountPerSupplier*numSuppliersPerSession + 1)
	// Increase the number of relay such that the supplier did "free work" and would
	// be able to claim more than the max claimable amount.
	numRelays *= 5
	numTokensClaimed := getNumTokensClaimed(
		numRelays,
		serviceComputeUnitsPerRelay,
		globalComputeUnitsToTokensMultiplier,
		globalComputeUnitCostGranularity,
	)

	// Retrieve the app and supplier module addresses
	appModuleAddress := authtypes.NewModuleAddress(apptypes.ModuleName).String()
	supplierModuleAddress := authtypes.NewModuleAddress(suppliertypes.ModuleName).String()

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	sharedParams.ComputeUnitsToTokensMultiplier = globalComputeUnitsToTokensMultiplier
	err := keepers.SharedKeeper.SetParams(ctx, sharedParams)
	require.NoError(t, err)

	// Setting inflation to zero so we are testing the BurnEqualsMint logic exclusively.
	tokenomicsParams := keepers.Keeper.GetParams(ctx)
	tokenomicsParams.GlobalInflationPerClaim = 0
	tokenomicsParams.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Dao:         0,
		Proposer:    0,
		Supplier:    1,
		SourceOwner: 0,
		Application: 0,
	}
	err = keepers.Keeper.SetParams(ctx, tokenomicsParams)
	require.NoError(t, err)

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(pocket.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddressBech32()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}
	services := []*sharedtypes.SupplierServiceConfig{{
		ServiceId: service.Id,
		RevShare:  supplierRevShares,
	}}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierInitialStake)
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplierRevShares[0].Address,
		services, 1, 0,
	)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:         supplierRevShares[0].Address,
		OperatorAddress:      supplierRevShares[0].Address,
		Stake:                &supplierStake,
		Services:             services,
		ServiceConfigHistory: serviceConfigHistory,
	}
	keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	// Query the account and module start balances
	appStartBalance := getBalance(t, ctx, keepers, app.GetAddress())
	appModuleStartBalance := getBalance(t, ctx, keepers, appModuleAddress)
	supplierModuleStartBalance := getBalance(t, ctx, keepers, supplierModuleAddress)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)
	pendingResult := tlm.NewClaimSettlementResult(claim)

	settlementContext := tokenomicskeeper.NewSettlementContext(
		ctx,
		keepers.Keeper,
		keepers.Logger(),
	)

	err = settlementContext.ClaimCacheWarmUp(ctx, &claim)
	require.NoError(t, err)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
	require.NoError(t, err)

	// Execute the pending results
	pendingResults := make(tlm.ClaimSettlementResults, 0)
	pendingResults.Append(pendingResult)
	err = keepers.ExecutePendingSettledResults(cosmostypes.UnwrapSDKContext(ctx), pendingResults)
	require.NoError(t, err)

	// Persist the actors state
	settlementContext.FlushAllActorsToStore(ctx)

	// Assert that `applicationAddress` account balance is *unchanged*
	appEndBalance := getBalance(t, ctx, keepers, app.GetAddress())
	require.EqualValues(t, appStartBalance, appEndBalance)

	// Determine the expected app end stake amount and the expected app burn
	appBurn := cosmosmath.NewInt(maxClaimableAmountPerSupplier)
	appBurnCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, appBurn)
	expectedAppEndStakeAmount := appInitialStake.Sub(appBurn)

	// Assert that `applicationAddress` staked balance has decreased by the max claimable amount
	app, appIsFound := keepers.GetApplication(ctx, app.GetAddress())
	actualAppEndStakeAmount := app.GetStake().Amount
	require.True(t, appIsFound)
	require.Equal(t, expectedAppEndStakeAmount, actualAppEndStakeAmount)

	// Sanity
	require.Less(t, maxClaimableAmountPerSupplier, numTokensClaimed)

	// Assert that app module balance is *decreased* by the appropriate amount
	// DEV_NOTE: The application module account burns the amount of uPOKT that was held in escrow
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
	// DEV_NOTE: Supplier rewards are minted to the supplier module account but then immediately
	// distributed to the supplier accounts which provided service in a given session.
	supplierModuleEndBalance := getBalance(t, ctx, keepers, supplierModuleAddress)
	require.EqualValues(t, supplierModuleStartBalance, supplierModuleEndBalance)

	// Assert that the supplier shareholders account balances have *increased* by
	// the appropriate amount w.r.t token distribution.
	// The supplier gets a percentage of the total settlement based on MintEqualsBurnClaimDistribution
	supplierAllocation := appBurn.MulRaw(int64(keepers.Keeper.GetParams(ctx).MintEqualsBurnClaimDistribution.Supplier * 100)).QuoRaw(100)
	shareAmounts := tlm.GetShareAmountMap(supplierRevShares, supplierAllocation)
	for shareHolderAddr, expectedShareAmount := range shareAmounts {
		shareHolderBalance := getBalance(t, ctx, keepers, shareHolderAddr)
		require.Equal(t, expectedShareAmount, shareHolderBalance.Amount)
	}

	// Check that the expected burn >> effective burn because application is overserviced

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	appOverservicedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventApplicationOverserviced](t, events)
	require.Len(t, appOverservicedEvents, 1, "unexpected number of event overserviced events")
	appOverservicedEvent := appOverservicedEvents[0]

	require.Equal(t, app.GetAddress(), appOverservicedEvent.ApplicationAddr)
	require.Equal(t, supplier.GetOperatorAddress(), appOverservicedEvent.SupplierOperatorAddr)
	// Parse the string representations of the coins to compare amounts
	expectedBurnCoin, err := cosmostypes.ParseCoinNormalized(appOverservicedEvent.ExpectedBurn)
	require.NoError(t, err)
	require.Equal(t, numTokensClaimed, expectedBurnCoin.Amount.Int64())

	effectiveBurnCoin, err := cosmostypes.ParseCoinNormalized(appOverservicedEvent.EffectiveBurn)
	require.NoError(t, err)
	require.Equal(t, appBurn, effectiveBurnCoin.Amount)
	require.Less(t, appBurn.Int64(), numTokensClaimed)
}

func TestProcessTokenLogicModules_TLMGlobalMint_Valid_MintDistributionCorrect(t *testing.T) {
	// Test Parameters
	appInitialStake := apptypes.DefaultMinStake.Amount.Mul(cosmosmath.NewInt(2))
	supplierInitialStake := cosmosmath.NewInt(1000000)
	supplierRevShareRatios := []uint64{12, 38, 50}
	globalComputeUnitCostGranularity := uint64(1000000)
	globalComputeUnitsToTokensMultiplier := uint64(1) * globalComputeUnitCostGranularity
	serviceComputeUnitsPerRelay := uint64(1)
	service := prepareTestService(serviceComputeUnitsPerRelay)
	numRelays := uint64(1000) // By supplier for application in this session
	numTokensClaimed := getNumTokensClaimed(
		numRelays,
		serviceComputeUnitsPerRelay,
		globalComputeUnitsToTokensMultiplier,
		globalComputeUnitCostGranularity,
	)
	numTokensClaimedInt := cosmosmath.NewIntFromUint64(uint64(numTokensClaimed))
	proposerConsAddr := sample.ConsAddress()
	proposerValOperatorAddr := sample.ValOperatorAddress()
	daoAddress := authtypes.NewModuleAddress(govtypes.ModuleName)

	tokenLogicModules := tlm.NewDefaultTokenLogicModules()

	// Prepare the keepers
	opts := []testkeeper.TokenomicsModuleKeepersOptFn{
		testkeeper.WithService(*service),
		testkeeper.WithBlockProposer(proposerConsAddr, proposerValOperatorAddr),
		testkeeper.WithTokenLogicModules(tokenLogicModules),
		testkeeper.WithDefaultModuleBalances(),
	}
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, opts...)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(ctx, *service)

	// Set the dao_reward_address param on the tokenomics keeper.
	tokenomicsParams := keepers.Keeper.GetParams(ctx)
	tokenomicsParams.DaoRewardAddress = daoAddress.String()
	keepers.Keeper.SetParams(ctx, tokenomicsParams)

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	sharedParams.ComputeUnitsToTokensMultiplier = globalComputeUnitsToTokensMultiplier
	err := keepers.SharedKeeper.SetParams(ctx, sharedParams)
	require.NoError(t, err)

	// Add a new application with non-zero app stake end balance to assert against.
	appStake := cosmostypes.NewCoin(pocket.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Prepare the supplier revenue shares
	supplierRevShares := make([]*sharedtypes.ServiceRevenueShare, len(supplierRevShareRatios))
	for i := range supplierRevShares {
		shareHolderAddress := sample.AccAddressBech32()
		supplierRevShares[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareHolderAddress,
			RevSharePercentage: supplierRevShareRatios[i],
		}
	}
	services := []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id, RevShare: supplierRevShares}}

	// Add a new supplier.
	supplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierInitialStake)
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplierRevShares[0].Address,
		services, 1, 0,
	)
	supplier := sharedtypes.Supplier{
		// Make the first shareholder the supplier itself.
		OwnerAddress:         supplierRevShares[0].Address,
		OperatorAddress:      supplierRevShares[0].Address,
		Stake:                &supplierStake,
		Services:             services,
		ServiceConfigHistory: serviceConfigHistory,
	}
	keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	// Prepare the claim for which the supplier did work for the application
	claim := prepareTestClaim(numRelays, service, &app, &supplier)
	pendingResult := tlm.NewClaimSettlementResult(claim)

	// Prepare addresses
	appAddress := app.Address

	// Determine balances before inflation
	daoBalanceBefore := getBalance(t, ctx, keepers, daoAddress.String())
	serviceOwnerBalanceBefore := getBalance(t, ctx, keepers, service.OwnerAddress)
	appBalanceBefore := getBalance(t, ctx, keepers, appAddress)
	supplierShareholderBalancesBeforeSettlementMap := make(map[string]*cosmostypes.Coin, len(supplierRevShares))
	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		supplierShareholderBalancesBeforeSettlementMap[addr] = getBalance(t, ctx, keepers, addr)
	}

	settlementContext := tokenomicskeeper.NewSettlementContext(
		ctx,
		keepers.Keeper,
		keepers.Logger(),
	)

	err = settlementContext.ClaimCacheWarmUp(ctx, &claim)
	require.NoError(t, err)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
	require.NoError(t, err)

	// Persist the actors state
	settlementContext.FlushAllActorsToStore(ctx)

	// Execute the pending results
	pendingResults := make(tlm.ClaimSettlementResults, 0)
	pendingResults.Append(pendingResult)
	err = keepers.ExecutePendingSettledResults(cosmostypes.UnwrapSDKContext(ctx), pendingResults)
	require.NoError(t, err)

	// Determine balances after inflation
	daoBalanceAfter := getBalance(t, ctx, keepers, daoAddress.String())
	serviceOwnerBalanceAfter := getBalance(t, ctx, keepers, service.OwnerAddress)
	appBalanceAfter := getBalance(t, ctx, keepers, appAddress)
	supplierShareholderBalancesAfter := make(map[string]*cosmostypes.Coin, len(supplierRevShares))
	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		supplierShareholderBalancesAfter[addr] = getBalance(t, ctx, keepers, addr)
	}

	// Compute the expected amount to mint.
	globalInflationPerClaimRat, err := encoding.Float64ToRat(tokenomicsParams.GlobalInflationPerClaim)
	require.NoError(t, err)

	numTokensClaimedRat := new(big.Rat).SetInt(numTokensClaimedInt.BigInt())
	numTokensMintedRat := new(big.Rat).Mul(numTokensClaimedRat, globalInflationPerClaimRat)
	reminder := new(big.Int)
	numTokensMintedInt, reminder := new(big.Int).QuoRem(
		numTokensMintedRat.Num(),
		numTokensMintedRat.Denom(),
		reminder,
	)

	// Ceil the number of tokens minted if there is a remainder.
	if reminder.Cmp(big.NewInt(0)) != 0 {
		numTokensMintedInt = numTokensMintedInt.Add(numTokensMintedInt, big.NewInt(1))
	}
	numTokensMinted := cosmosmath.NewIntFromBigInt(numTokensMintedInt)

	// Compute the expected amount minted to each module from Global Mint TLM.
	propMintFromGlobalMint := computeShare(t, numTokensMintedRat, tokenomicsParams.MintAllocationPercentages.Proposer)
	serviceOwnerMintFromGlobalMint := computeShare(t, numTokensMintedRat, tokenomicsParams.MintAllocationPercentages.SourceOwner)
	appMintFromGlobalMint := computeShare(t, numTokensMintedRat, tokenomicsParams.MintAllocationPercentages.Application)
	supplierMintFromGlobalMint := computeShare(t, numTokensMintedRat, tokenomicsParams.MintAllocationPercentages.Supplier)
	// The DAO mint gets any remainder resulting from integer division.
	daoMintFromGlobalMint := numTokensMinted.Sub(propMintFromGlobalMint).Sub(serviceOwnerMintFromGlobalMint).Sub(appMintFromGlobalMint).Sub(supplierMintFromGlobalMint)

	// Compute the expected amount from Relay Burn Equals Mint TLM distribution.
	settlementAmount := numTokensClaimedInt
	propDistributionFromBurnEqualsMint := settlementAmount.MulRaw(int64(tokenomicsParams.MintEqualsBurnClaimDistribution.Proposer * 100)).QuoRaw(100)
	serviceOwnerDistributionFromBurnEqualsMint := settlementAmount.MulRaw(int64(tokenomicsParams.MintEqualsBurnClaimDistribution.SourceOwner * 100)).QuoRaw(100)
	appDistributionFromBurnEqualsMint := settlementAmount.MulRaw(int64(tokenomicsParams.MintEqualsBurnClaimDistribution.Application * 100)).QuoRaw(100)
	supplierDistributionFromBurnEqualsMint := settlementAmount.MulRaw(int64(tokenomicsParams.MintEqualsBurnClaimDistribution.Supplier * 100)).QuoRaw(100)
	// The DAO gets the remainder to ensure all settlement tokens are distributed
	daoDistributionFromBurnEqualsMint := settlementAmount.Sub(propDistributionFromBurnEqualsMint).Sub(serviceOwnerDistributionFromBurnEqualsMint).Sub(appDistributionFromBurnEqualsMint).Sub(supplierDistributionFromBurnEqualsMint)

	// Total expected amounts from both TLMs.
	propTotalExpected := propMintFromGlobalMint.Add(propDistributionFromBurnEqualsMint)
	serviceOwnerTotalExpected := serviceOwnerMintFromGlobalMint.Add(serviceOwnerDistributionFromBurnEqualsMint)
	appTotalExpected := appMintFromGlobalMint.Add(appDistributionFromBurnEqualsMint)
	daoTotalExpected := daoMintFromGlobalMint.Add(daoDistributionFromBurnEqualsMint).Add(numTokensMinted)

	// Verify that ModToAcctTransfer operations include validator rewards using ModToAcctTransfer
	modToAcctTransfers := pendingResult.GetModToAcctTransfers()
	validatorRewardsFound := false
	totalValidatorRewardAmount := cosmosmath.ZeroInt()

	// Check for validator commission and delegator reward transfers
	for _, transfer := range modToAcctTransfers {
		if transfer.OpReason == tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_PROPOSER_REWARD_DISTRIBUTION ||
			transfer.OpReason == tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION ||
			transfer.OpReason == tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_PROPOSER_REWARD_DISTRIBUTION ||
			transfer.OpReason == tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION {
			validatorRewardsFound = true
			totalValidatorRewardAmount = totalValidatorRewardAmount.Add(transfer.Coin.Amount)
		}
	}

	require.True(t, validatorRewardsFound, "Should find ModToAcctTransfer operations for validator/delegator rewards")
	require.Equal(t, propTotalExpected, totalValidatorRewardAmount, "Total validator reward amount should match expected proposer allocation")
	require.Equal(t, serviceOwnerBalanceBefore.Amount.Add(serviceOwnerTotalExpected), serviceOwnerBalanceAfter.Amount)
	require.Equal(t, appBalanceBefore.Amount.Add(appTotalExpected), appBalanceAfter.Amount)
	require.Equal(t, daoBalanceBefore.Amount.Add(daoTotalExpected), daoBalanceAfter.Amount)

	supplierMintRat := new(big.Rat).SetInt(supplierMintFromGlobalMint.BigInt())
	supplierDistributionRat := new(big.Rat).SetInt(supplierDistributionFromBurnEqualsMint.BigInt())
	shareHoldersBalancesAfterSettlementMap := make(map[string]cosmosmath.Int, len(supplierRevShares))
	supplierMintWithoutRemainder := cosmosmath.NewInt(0)
	supplierDistributionWithoutRemainder := cosmosmath.NewInt(0)
	for _, revShare := range supplierRevShares {
		addr := revShare.Address

		// Compute the expected balance increase for the shareholder
		mintShareFloat := float64(revShare.RevSharePercentage) / 100.0
		// From Relay Burn Equals Mint TLM distribution
		distributionShare := computeShare(t, supplierDistributionRat, mintShareFloat)
		// From Global Mint TLM distribution
		mintShare := computeShare(t, supplierMintRat, mintShareFloat)
		balanceIncrease := distributionShare.Add(mintShare)

		// Compute the expected balance after minting
		balanceBefore := supplierShareholderBalancesBeforeSettlementMap[addr]
		shareHoldersBalancesAfterSettlementMap[addr] = balanceBefore.Amount.Add(balanceIncrease)

		supplierMintWithoutRemainder = supplierMintWithoutRemainder.Add(mintShare)
		supplierDistributionWithoutRemainder = supplierDistributionWithoutRemainder.Add(distributionShare)
	}

	// The first shareholder gets any remainder resulting from integer division.
	firstShareHolderAddr := supplierRevShares[0].Address
	firstShareHolderBalance := shareHoldersBalancesAfterSettlementMap[firstShareHolderAddr]
	mintRemainder := supplierMintFromGlobalMint.Sub(supplierMintWithoutRemainder)
	distributionRemainder := supplierDistributionFromBurnEqualsMint.Sub(supplierDistributionWithoutRemainder)
	totalRemainder := mintRemainder.Add(distributionRemainder)
	shareHoldersBalancesAfterSettlementMap[firstShareHolderAddr] = firstShareHolderBalance.Add(totalRemainder)

	for _, revShare := range supplierRevShares {
		addr := revShare.Address
		balanceAfter := supplierShareholderBalancesAfter[addr].Amount
		expectedBalanceAfter := shareHoldersBalancesAfterSettlementMap[addr]
		require.Equal(t, expectedBalanceAfter, balanceAfter)
	}

	foundApp, appFound := keepers.GetApplication(ctx, appAddress)
	require.True(t, appFound)

	appStakeAfter := foundApp.GetStake().Amount
	expectedStakeAfter := appInitialStake.Sub(numTokensMinted).Sub(numTokensClaimedInt)
	require.Equal(t, expectedStakeAfter, appStakeAfter)
}

func TestProcessTokenLogicModules_AppNotFound(t *testing.T) {
	keeper, ctx, _, supplierOperatorAddr, service := testkeeper.TokenomicsKeeperWithActorAddrs(t)

	// The base claim whose root will be customized for testing purposes
	numRelays := uint64(42)
	numComputeUnits := numRelays * service.ComputeUnitsPerRelay
	claim := prooftypes.Claim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      sample.AccAddressBech32(), // Random address
			ServiceId:               service.Id,
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSumAndCount(numComputeUnits, numRelays),
	}
	pendingResult := tlm.NewClaimSettlementResult(claim)

	settlementContext := tokenomicskeeper.NewSettlementContext(ctx, &keeper, keeper.Logger())

	// Ignoring the error from ClaimCacheWarmUp as it will short-circuit the test
	// and we want to test the error from ProcessTokenLogicModules.
	_ = settlementContext.ClaimCacheWarmUp(ctx, &claim)

	// Process the token logic modules
	err := keeper.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
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
	pendingResult := tlm.NewClaimSettlementResult(claim)

	settlementContext := tokenomicskeeper.NewSettlementContext(ctx, &keeper, keeper.Logger())

	// Ignoring the error from ClaimCacheWarmUp as it will short-circuit the test
	// and we want to test the error from ProcessTokenLogicModules.
	_ = settlementContext.ClaimCacheWarmUp(ctx, &claim)

	// Execute test function
	err := keeper.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
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
				// A root with all 'a's is a valid value since each of the hash, sum and size
				// will be []byte{0x61, 0x61, ...} with their respective sizes.
				// The current test suite sets the CUPR to 1, making sum == count * CUPR
				// valid. So, we can change the last byte to 'b' to make it invalid.
				root := bytes.Repeat([]byte("a"), protocol.TrieRootSize)
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
			pendingResult := tlm.NewClaimSettlementResult(claim)

			settlementContext := tokenomicskeeper.NewSettlementContext(ctx, &keeper, keeper.Logger())

			// Ignoring the error from ClaimCacheWarmUp as it will short-circuit the test
			// and we want to test the error from ProcessTokenLogicModules.
			_ = settlementContext.ClaimCacheWarmUp(ctx, &claim)

			// Execute test function
			err := keeper.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)

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
		claim       prooftypes.Claim
		errExpected bool
		expectErr   error
	}{

		{
			desc: "Valid claim",
			claim: func() prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				return claim
			}(),
			errExpected: false,
		},
		{
			desc: "claim with nil session header",
			claim: func() prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader = nil
				return claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsClaimSessionHeaderNil,
		},
		{
			desc: "claim with invalid session id",
			claim: func() prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader.SessionId = ""
				return claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsClaimSessionHeaderInvalid,
		},
		{
			desc: "claim with invalid application address",
			claim: func() prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SessionHeader.ApplicationAddress = "invalid address"
				return claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsClaimSessionHeaderInvalid,
		},
		{
			desc: "claim with invalid supplier operator address",
			claim: func() prooftypes.Claim {
				claim := testproof.BaseClaim(service.Id, appAddr, supplierOperatorAddr, numRelays)
				claim.SupplierOperatorAddress = "invalid address"
				return claim
			}(),
			errExpected: true,
			expectErr:   tokenomicstypes.ErrTokenomicsSupplierNotFound,
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
				pendingResult := tlm.NewClaimSettlementResult(test.claim)

				settlementContext := tokenomicskeeper.NewSettlementContext(ctx, &keeper, keeper.Logger())

				// Ignoring the error from ClaimCacheWarmUp as it will short-circuit the test
				// and we want to test the error from ProcessTokenLogicModules.
				_ = settlementContext.ClaimCacheWarmUp(ctx, &test.claim)
				return keeper.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
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

// TestProcessTokenLogicModules_MultipleValidators tests that both RelayBurnEqualsMint and GlobalMint TLMs
// properly distribute rewards to multiple validators with multiple delegators each, proportionally based on staking weight.
func TestProcessTokenLogicModules_MultipleValidators(t *testing.T) {
	// Test Parameters
	appInitialStake := apptypes.DefaultMinStake.Amount.Mul(cosmosmath.NewInt(2))
	supplierInitialStake := cosmosmath.NewInt(1000000)
	globalComputeUnitCostGranularity := uint64(1000000)
	globalComputeUnitsToTokensMultiplier := uint64(1) * globalComputeUnitCostGranularity
	numRelays := uint64(10000)

	// Create service
	service := prepareTestService(1)
	daoAddress := sample.AccAddressBech32()

	// Create validators with deterministic addresses for precise testing
	// Total stake: 1,000,000 tokens distributed as 60%, 30%, 10%
	validator1OperAddr := sample.ValOperatorAddress().String() // 60% stake, 5% commission
	validator2OperAddr := sample.ValOperatorAddress().String() // 30% stake, 10% commission
	validator3OperAddr := sample.ValOperatorAddress().String() // 10% stake, 15% commission

	// Convert validator operator addresses to account addresses (for reward distribution)
	validator1AccAddr := cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator1OperAddr)).String()
	validator2AccAddr := cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator2OperAddr)).String()
	validator3AccAddr := cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator3OperAddr)).String()

	// Create delegator addresses using pre-generated accounts (matching mock staking keeper logic)
	// The mock creates 2 delegators per validator using indices starting at 20:
	// - Validator 0: delegators 20, 21
	// - Validator 1: delegators 22, 23
	// - Validator 2: delegators 24, 25
	delegator1_1 := testkeyring.MustPreGeneratedAccountAtIndex(20).Address.String() // First delegator for validator1
	delegator1_2 := testkeyring.MustPreGeneratedAccountAtIndex(21).Address.String() // Second delegator for validator1
	delegator2_1 := testkeyring.MustPreGeneratedAccountAtIndex(22).Address.String() // First delegator for validator2
	delegator2_2 := testkeyring.MustPreGeneratedAccountAtIndex(23).Address.String() // Second delegator for validator2
	delegator3_1 := testkeyring.MustPreGeneratedAccountAtIndex(24).Address.String() // First delegator for validator3
	delegator3_2 := testkeyring.MustPreGeneratedAccountAtIndex(25).Address.String() // Second delegator for validator3

	validators := []stakingtypes.Validator{
		{
			OperatorAddress: validator1OperAddr,
			Tokens:          cosmosmath.NewInt(600000), // 60% of total stake
			Status:          stakingtypes.Bonded,
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate: cosmosmath.LegacyNewDecWithPrec(5, 2), // 5% commission
				},
			},
			DelegatorShares: cosmosmath.LegacyNewDecFromInt(cosmosmath.NewInt(600000)),
		},
		{
			OperatorAddress: validator2OperAddr,
			Tokens:          cosmosmath.NewInt(300000), // 30% of total stake
			Status:          stakingtypes.Bonded,
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate: cosmosmath.LegacyNewDecWithPrec(10, 2), // 10% commission
				},
			},
			DelegatorShares: cosmosmath.LegacyNewDecFromInt(cosmosmath.NewInt(300000)),
		},
		{
			OperatorAddress: validator3OperAddr,
			Tokens:          cosmosmath.NewInt(100000), // 10% of total stake
			Status:          stakingtypes.Bonded,
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate: cosmosmath.LegacyNewDecWithPrec(15, 2), // 15% commission
				},
			},
			DelegatorShares: cosmosmath.LegacyNewDecFromInt(cosmosmath.NewInt(100000)),
		},
	}

	// Set up tokenomics keepers with our custom validators
	opts := []testkeeper.TokenomicsModuleKeepersOptFn{
		testkeeper.WithService(*service),
		testkeeper.WithTokenLogicModules(tlm.NewDefaultTokenLogicModules()),
		testkeeper.WithDefaultModuleBalances(),
		testkeeper.WithValidators(validators), // Much simpler than complex mock injection!
	}
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, opts...)

	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(ctx, *service)

	// Set the dao_reward_address param on the tokenomics keeper
	tokenomicsParams := keepers.Keeper.GetParams(ctx)
	tokenomicsParams.DaoRewardAddress = daoAddress
	// Enable validator rewards for both TLMs
	tokenomicsParams.MintAllocationPercentages.Proposer = 0.1       // 10% inflation goes to all validators (GlobalMint TLM)
	tokenomicsParams.MintEqualsBurnClaimDistribution.Proposer = 0.1 // 10% settlement goes to all validators (RelayBurnEqualsMint TLM)
	err := keepers.Keeper.SetParams(ctx, tokenomicsParams)
	require.NoError(t, err)

	// Set compute_units_to_tokens_multiplier to simplify expectation calculations
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	sharedParams.ComputeUnitsToTokensMultiplier = globalComputeUnitsToTokensMultiplier
	err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
	require.NoError(t, err)

	// Add application
	appStake := cosmostypes.NewCoin(pocket.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(ctx, app)

	// Add supplier
	supplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierInitialStake)
	supplierAddr := sample.AccAddressBech32()
	supplierServices := []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: service.Id,
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: supplierAddr, RevSharePercentage: 100},
			},
		},
	}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplierAddr, supplierServices, 1, 0,
	)
	supplier := sharedtypes.Supplier{
		OwnerAddress:         supplierAddr,
		OperatorAddress:      supplierAddr,
		Stake:                &supplierStake,
		Services:             supplierServices,
		ServiceConfigHistory: serviceConfigHistory,
	}
	keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	// Prepare claim and process TLMs
	claim := prepareTestClaim(numRelays, service, &app, &supplier)
	pendingResult := tlm.NewClaimSettlementResult(claim)

	settlementContext := tokenomicskeeper.NewSettlementContext(
		ctx,
		keepers.Keeper,
		keepers.Logger(),
	)

	err = settlementContext.ClaimCacheWarmUp(ctx, &claim)
	require.NoError(t, err)

	// Process the token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, settlementContext, pendingResult)
	require.NoError(t, err)

	// Persist the actors state
	settlementContext.FlushAllActorsToStore(ctx)

	// Execute the pending results
	pendingResults := make(tlm.ClaimSettlementResults, 0)
	pendingResults.Append(pendingResult)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	err = keepers.ExecutePendingSettledResults(sdkCtx, pendingResults)
	require.NoError(t, err)

	// Verify that rewards were distributed to ALL validators proportionally
	// The mock staking keeper provides 3 validators with different stakes and commission rates.
	// The distribution logic in distributeRewardsToAllValidatorsAndDelegatesByStakeWeight()
	// calls GetBondedValidatorsByPower() to get ALL validators and distributes proportionally
	// based on their stake weight, then distributes to each validator's delegators after commission.

	// Process and track reward distribution
	modToAcctTransfers := pendingResult.GetModToAcctTransfers()
	distribution := newRewardDistribution()

	for _, transfer := range modToAcctTransfers {
		distribution.processTransfer(transfer)
	}

	// Calculate expected reward distributions
	// Total settlement: 10,000 relays × 1 CU/relay × 1 uPOKT/CU = 10,000 uPOKT
	// RelayBurnEqualsMint: 10,000 × 10% proposer = 1,000 uPOKT to validators
	// GlobalMint: 10,000 × 10% inflation × 10% proposer = 100 uPOKT to validators
	// Total validator rewards: 1,000 + 100 = 1,100 uPOKT

	// Distribution by stake weight (60%, 30%, 10%):
	// - Validator 1: 1,100 × 60% = 660 uPOKT → 660 × 5% = 33 commission, 627 to delegators (314+313)
	// - Validator 2: 1,100 × 30% = 330 uPOKT → 330 × 10% = 33 commission, 297 to delegators (149+148)
	// - Validator 3: 1,100 × 10% = 110 uPOKT → 110 × 15% = 16 commission, 94 to delegators (46+48)

	expectedValidatorCommission := cosmosmath.NewInt(82) // 33 + 33 + 16
	expectedDelegatorRewards := cosmosmath.NewInt(1018)  // 627 + 297 + 94
	expectedTotalRewards := cosmosmath.NewInt(1100)      // Total distributed

	// Verify exact transfer counts
	// 3 validators get rewards from each TLM = 6 validator transfers
	// 3 validators × 2 delegators each × 2 TLMs = 12 delegator transfers
	require.Equal(t, 3, distribution.RelayBurnValidatorCount, "RelayBurnEqualsMint validator rewards")
	require.Equal(t, 6, distribution.RelayBurnDelegatorCount, "RelayBurnEqualsMint delegator rewards")
	require.Equal(t, 3, distribution.GlobalMintValidatorCount, "GlobalMint validator rewards")
	require.Equal(t, 6, distribution.GlobalMintDelegatorCount, "GlobalMint delegator rewards")

	// Verify total amounts
	totalValidatorRewards := distribution.getTotalValidatorRewards()
	totalDelegatorRewardAmount := distribution.getTotalDelegatorRewards()

	require.Equal(t, expectedValidatorCommission, totalValidatorRewards,
		"total validator commission should be exactly %s uPOKT", expectedValidatorCommission)
	require.Equal(t, expectedDelegatorRewards, totalDelegatorRewardAmount,
		"total delegator rewards should be exactly %s uPOKT", expectedDelegatorRewards)
	require.Equal(t, expectedTotalRewards, totalValidatorRewards.Add(totalDelegatorRewardAmount),
		"total distributed should equal total rewards")

	// Verify distribution pattern
	require.Equal(t, 3, len(distribution.ValidatorRewards), "should have exactly 3 validators")
	require.Equal(t, 6, len(distribution.DelegatorRewards), "should have exactly 6 delegators")

	// Verify precise validator commission amounts
	require.Equal(t, int64(33), distribution.ValidatorRewards[validator1AccAddr].Int64(),
		"Validator1 (60% stake, 5% commission) should receive exactly 33 uPOKT commission")
	require.Equal(t, int64(33), distribution.ValidatorRewards[validator2AccAddr].Int64(),
		"Validator2 (30% stake, 10% commission) should receive exactly 33 uPOKT commission")
	require.Equal(t, int64(16), distribution.ValidatorRewards[validator3AccAddr].Int64(),
		"Validator3 (10% stake, 15% commission) should receive exactly 16 uPOKT commission")

	// Verify precise delegator reward amounts
	// Validator1's delegators split 627 uPOKT (660 - 33 commission)
	require.Equal(t, int64(314), distribution.DelegatorRewards[delegator1_1].Int64(),
		"Delegator1_1 should receive 314 uPOKT")
	require.Equal(t, int64(313), distribution.DelegatorRewards[delegator1_2].Int64(),
		"Delegator1_2 should receive 313 uPOKT")

	// Validator2's delegators split 297 uPOKT (330 - 33 commission)
	require.Equal(t, int64(149), distribution.DelegatorRewards[delegator2_1].Int64(),
		"Delegator2_1 should receive 149 uPOKT")
	require.Equal(t, int64(148), distribution.DelegatorRewards[delegator2_2].Int64(),
		"Delegator2_2 should receive 148 uPOKT")

	// Validator3's delegators split 94 uPOKT (110 - 16 commission)
	// Note: Due to rounding, the split is 46+48=94 rather than 47+47
	require.Equal(t, int64(46), distribution.DelegatorRewards[delegator3_1].Int64(),
		"Delegator3_1 should receive 46 uPOKT")
	require.Equal(t, int64(48), distribution.DelegatorRewards[delegator3_2].Int64(),
		"Delegator3_2 should receive 48 uPOKT")

	// Verify total delegator rewards
	actualDelegatorRewards := distribution.getTotalDelegatorRewards()
	require.Equal(t, expectedDelegatorRewards, actualDelegatorRewards,
		"total delegator rewards should be exactly %s uPOKT", expectedDelegatorRewards)

	t.Logf("✓ Distributed %s uPOKT total: %s to validators, %s to delegators",
		expectedTotalRewards, totalValidatorRewards, actualDelegatorRewards)
	t.Logf("  RelayBurn: %d validators, %d delegators | GlobalMint: %d validators, %d delegators",
		distribution.RelayBurnValidatorCount, distribution.RelayBurnDelegatorCount,
		distribution.GlobalMintValidatorCount, distribution.GlobalMintDelegatorCount)
}

func TestProcessTokenLogicModules_AppStakeInsufficientToCoverGlobalInflationAmount(t *testing.T) {
	t.Skip("TODO_TEST: Test application stake that is insufficient to cover the global inflation amount, for reimbursment and the max claim should scale down proportionally")
}

func TestProcessTokenLogicModules_AppStakeTooLowRoundingToZero(t *testing.T) {
	t.Skip("TODO_TEST: Test application stake that is too low which results in stake/num_suppliers rounding down to zero")
}

func TestProcessTokenLogicModules_AppStakeDropsBelowMinStakeAfterSession(t *testing.T) {
	t.Skip("TODO_TEST: Test that application stake being auto-unbonding after the stake drops below the required minimum when settling session accounting")
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
		OwnerAddress:         sample.AccAddressBech32(),
	}
}

func getBalance(
	t *testing.T,
	ctx context.Context,
	bankKeeper tokenomicstypes.BankKeeper,
	addr string,
) *cosmostypes.Coin {
	appBalanceRes, err := bankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: addr,
		Denom:   "upokt",
	})
	require.NoError(t, err)

	balance := appBalanceRes.GetBalance()
	require.NotNil(t, balance)

	return balance
}

// computeShare computes the share of the given amount based a percentage.
func computeShare(t *testing.T, amount *big.Rat, sharePercentage float64) cosmosmath.Int {
	amountRat, err := encoding.Float64ToRat(sharePercentage)
	require.NoError(t, err)

	mintRat := new(big.Rat).Mul(amount, amountRat)
	flooredShare := new(big.Int).Quo(mintRat.Num(), mintRat.Denom())

	return cosmosmath.NewIntFromBigInt(flooredShare)
}

// getNumTokensClaimed calculates the number of tokens claimed
func getNumTokensClaimed(
	numRelays,
	serviceComputeUnitsPerRelay,
	computeUnitsToTokensMultiplier,
	computeUnitCostGranularity uint64,
) int64 {
	computeUnitCostUpokt := new(big.Rat).SetFrac64(
		int64(computeUnitsToTokensMultiplier),
		int64(computeUnitCostGranularity),
	)

	numComputeUnits := new(big.Rat).SetUint64(numRelays * serviceComputeUnitsPerRelay)

	numTokensClaimedRat := new(big.Rat).Mul(numComputeUnits, computeUnitCostUpokt)
	return numTokensClaimedRat.Num().Int64() / numTokensClaimedRat.Denom().Int64()
}

// rewardDistribution tracks rewards distributed to validators and delegators
type rewardDistribution struct {
	ValidatorRewards map[string]cosmosmath.Int // Address -> Amount
	DelegatorRewards map[string]cosmosmath.Int // Address -> Amount

	// Counters for each TLM type
	RelayBurnValidatorCount  int
	RelayBurnDelegatorCount  int
	GlobalMintValidatorCount int
	GlobalMintDelegatorCount int
}

// newRewardDistribution creates a new reward distribution tracker
func newRewardDistribution() *rewardDistribution {
	return &rewardDistribution{
		ValidatorRewards: make(map[string]cosmosmath.Int),
		DelegatorRewards: make(map[string]cosmosmath.Int),
	}
}

// processTransfer updates the reward distribution based on a ModToAcctTransfer
func (rd *rewardDistribution) processTransfer(transfer tokenomicstypes.ModToAcctTransfer) {
	switch transfer.OpReason {
	case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_PROPOSER_REWARD_DISTRIBUTION:
		rd.RelayBurnValidatorCount++
		rd.ValidatorRewards[transfer.RecipientAddress] = transfer.Coin.Amount

	case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION:
		rd.RelayBurnDelegatorCount++
		if _, exists := rd.DelegatorRewards[transfer.RecipientAddress]; !exists {
			rd.DelegatorRewards[transfer.RecipientAddress] = cosmosmath.NewInt(0)
		}
		rd.DelegatorRewards[transfer.RecipientAddress] = rd.DelegatorRewards[transfer.RecipientAddress].Add(transfer.Coin.Amount)

	case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_PROPOSER_REWARD_DISTRIBUTION:
		rd.GlobalMintValidatorCount++
		if _, exists := rd.ValidatorRewards[transfer.RecipientAddress]; !exists {
			rd.ValidatorRewards[transfer.RecipientAddress] = cosmosmath.NewInt(0)
		}
		rd.ValidatorRewards[transfer.RecipientAddress] = rd.ValidatorRewards[transfer.RecipientAddress].Add(transfer.Coin.Amount)

	case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION:
		rd.GlobalMintDelegatorCount++
		if _, exists := rd.DelegatorRewards[transfer.RecipientAddress]; !exists {
			rd.DelegatorRewards[transfer.RecipientAddress] = cosmosmath.NewInt(0)
		}
		rd.DelegatorRewards[transfer.RecipientAddress] = rd.DelegatorRewards[transfer.RecipientAddress].Add(transfer.Coin.Amount)
	}
}

// getTotalValidatorRewards returns the sum of all validator rewards
func (rd *rewardDistribution) getTotalValidatorRewards() cosmosmath.Int {
	total := cosmosmath.NewInt(0)
	for _, amount := range rd.ValidatorRewards {
		total = total.Add(amount)
	}
	return total
}

// getTotalDelegatorRewards returns the sum of all delegator rewards
func (rd *rewardDistribution) getTotalDelegatorRewards() cosmosmath.Int {
	total := cosmosmath.NewInt(0)
	for _, amount := range rd.DelegatorRewards {
		total = total.Add(amount)
	}
	return total
}

func TestProcessTokenLogicModules_TLMBurnEqualsMint_Valid_WithRewardDistribution(t *testing.T) {
	// Test configuration constants
	const (
		// Initial stakes and helpers
		testApplicationStakeMultiplier = 2
		testSupplierInitialStakeUpokt  = 1000000

		// Tokenomics Governance Parameters
		testComputeUnitCostGranularity  = 1000000
		testServiceComputeUnitsPerRelay = 1
		testNumberOfRelaysInClaim       = 1000
		testGlobalInflationPerClaim     = 0.0 // Disable global inflation for this test

		// MintEqualsBurnClaimDistribution percentages
		testMintEqualsBurnDaoPercentage         = 0.24 // Increased to absorb proposer percentage
		testMintEqualsBurnProposerPercentage    = 0.0  // TODO: Re-enable to test distribution logic
		testMintEqualsBurnSupplierPercentage    = 0.73
		testMintEqualsBurnSourceOwnerPercentage = 0.03
		testMintEqualsBurnApplicationPercentage = 0.0

		// Supplier revenue share percentages (must add up to 100)
		testSupplierRevShareShareholder1Percentage = 12
		testSupplierRevShareShareholder2Percentage = 38
		testSupplierRevShareShareholder3Percentage = 50
	)

	// Prepare initial stake values
	testApplicationInitialStake := apptypes.DefaultMinStake.Amount.Mul(cosmosmath.NewInt(testApplicationStakeMultiplier))
	testSupplierInitialStake := cosmosmath.NewInt(testSupplierInitialStakeUpokt)

	// Setup test service
	testService := prepareTestService(testServiceComputeUnitsPerRelay)

	// Create proposer addresses for testing
	testProposerConsAddr := sample.ConsAddress()
	testProposerValOperAddr := sample.ValOperatorAddress()

	// Initialize blockchain keepers and context
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t,
		cosmoslog.NewNopLogger(),
		testkeeper.WithService(*testService),
		testkeeper.WithBlockProposer(testProposerConsAddr, testProposerValOperAddr),
		testkeeper.WithDefaultModuleBalances(),
	)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(ctx, *testService)

	// Validate claim is within relay mining bounds
	numSuppliersPerSession := int64(keepers.SessionKeeper.GetParams(ctx).NumSuppliersPerSession)
	testComputeUnitsToTokensMultiplier := uint64(1) * testComputeUnitCostGranularity
	totalTokensClaimedInSession := getNumTokensClaimed(
		testNumberOfRelaysInClaim,
		testServiceComputeUnitsPerRelay,
		testComputeUnitsToTokensMultiplier,
		testComputeUnitCostGranularity,
	)
	maxClaimableAmountPerSupplier := testApplicationInitialStake.Quo(cosmosmath.NewInt(numSuppliersPerSession))
	require.GreaterOrEqual(t, maxClaimableAmountPerSupplier.Int64(), totalTokensClaimedInSession)

	// Configure shared parameters for consistent token calculations
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	sharedParams.ComputeUnitsToTokensMultiplier = testComputeUnitsToTokensMultiplier
	err := keepers.SharedKeeper.SetParams(ctx, sharedParams)
	require.NoError(t, err)

	// Configure tokenomics parameters with specific reward distribution
	tokenomicsParams := keepers.Keeper.GetParams(ctx)
	tokenomicsParams.GlobalInflationPerClaim = testGlobalInflationPerClaim
	tokenomicsParams.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Dao:         testMintEqualsBurnDaoPercentage,
		Proposer:    testMintEqualsBurnProposerPercentage,
		Supplier:    testMintEqualsBurnSupplierPercentage,
		SourceOwner: testMintEqualsBurnSourceOwnerPercentage,
		Application: testMintEqualsBurnApplicationPercentage,
	}
	err = keepers.Keeper.SetParams(ctx, tokenomicsParams)
	require.NoError(t, err)

	// Create test application
	testApplicationStake := cosmostypes.NewCoin(pocket.DenomuPOKT, testApplicationInitialStake)
	testApplicationAddress := sample.AccAddressBech32()
	testApplication := apptypes.Application{
		Address:        testApplicationAddress,
		Stake:          &testApplicationStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: testService.Id}},
	}
	keepers.SetApplication(ctx, testApplication)

	// Create supplier revenue share configuration
	testSupplierRevSharePercentages := []uint64{
		testSupplierRevShareShareholder1Percentage,
		testSupplierRevShareShareholder2Percentage,
		testSupplierRevShareShareholder3Percentage,
	}
	supplierRevenueShareholders := make([]*sharedtypes.ServiceRevenueShare, len(testSupplierRevSharePercentages))
	for i := range supplierRevenueShareholders {
		shareholderAddress := sample.AccAddressBech32()
		supplierRevenueShareholders[i] = &sharedtypes.ServiceRevenueShare{
			Address:            shareholderAddress,
			RevSharePercentage: testSupplierRevSharePercentages[i],
		}
	}
	supplierServiceConfigs := []*sharedtypes.SupplierServiceConfig{{
		ServiceId: testService.Id,
		RevShare:  supplierRevenueShareholders,
	}}

	// Create test supplier
	testSupplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, testSupplierInitialStake)
	testSupplierOwnerAddress := supplierRevenueShareholders[0].Address
	testSupplierOperatorAddress := supplierRevenueShareholders[0].Address
	supplierServiceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		testSupplierOwnerAddress,
		supplierServiceConfigs, 1, 0,
	)
	testSupplier := sharedtypes.Supplier{
		OwnerAddress:         testSupplierOwnerAddress,
		OperatorAddress:      testSupplierOperatorAddress,
		Stake:                &testSupplierStake,
		Services:             supplierServiceConfigs,
		ServiceConfigHistory: supplierServiceConfigHistory,
	}
	keepers.SetAndIndexDehydratedSupplier(ctx, testSupplier)

	// Get addresses for balance verification
	// Convert the validator operator address to an account address for balance checks
	blockProposerAccountAddress := cosmostypes.AccAddress(testProposerValOperAddr).String()
	daoRewardAddress := tokenomicsParams.GetDaoRewardAddress()
	serviceSourceOwnerAddress := testService.OwnerAddress

	// Capture baseline balances for all actors before settlement
	daoBalanceBeforeSettlement := getBalance(t, ctx, keepers, daoRewardAddress)
	proposerBalanceBeforeSettlement := getBalance(t, ctx, keepers, blockProposerAccountAddress)
	sourceOwnerBalanceBeforeSettlement := getBalance(t, ctx, keepers, serviceSourceOwnerAddress)
	applicationBalanceBeforeSettlement := getBalance(t, ctx, keepers, testApplicationAddress)
	supplierShareholderBalancesBeforeSettlement := make(map[string]*cosmostypes.Coin)
	for _, shareholder := range supplierRevenueShareholders {
		supplierShareholderBalancesBeforeSettlement[shareholder.Address] = getBalance(t, ctx, keepers, shareholder.Address)
	}

	// Prepare claim and execute settlement
	testClaim := prepareTestClaim(testNumberOfRelaysInClaim, testService, &testApplication, &testSupplier)
	settlementResult := tlm.NewClaimSettlementResult(testClaim)
	settlementContext := tokenomicskeeper.NewSettlementContext(
		ctx,
		keepers.Keeper,
		keepers.Logger(),
	)
	err = settlementContext.ClaimCacheWarmUp(ctx, &testClaim)
	require.NoError(t, err)

	// Process token logic modules
	err = keepers.ProcessTokenLogicModules(ctx, settlementContext, settlementResult)
	require.NoError(t, err)

	// Execute settlement results
	pendingSettlementResults := make(tlm.ClaimSettlementResults, 0)
	pendingSettlementResults.Append(settlementResult)
	err = keepers.ExecutePendingSettledResults(cosmostypes.UnwrapSDKContext(ctx), pendingSettlementResults)
	require.NoError(t, err)

	// Calculate expected reward distributions from total settlement amount
	totalSettlementAmount := cosmosmath.NewInt(totalTokensClaimedInSession)
	expectedDaoRewardAmount := cosmosmath.NewInt(int64(float64(totalTokensClaimedInSession) * testMintEqualsBurnDaoPercentage))
	expectedProposerRewardAmount := cosmosmath.NewInt(int64(float64(totalTokensClaimedInSession) * testMintEqualsBurnProposerPercentage))
	expectedSupplierRewardAmount := cosmosmath.NewInt(int64(float64(totalTokensClaimedInSession) * testMintEqualsBurnSupplierPercentage))
	expectedSourceOwnerRewardAmount := cosmosmath.NewInt(int64(float64(totalTokensClaimedInSession) * testMintEqualsBurnSourceOwnerPercentage))
	expectedApplicationCostAmount := cosmosmath.NewInt(int64(float64(totalTokensClaimedInSession) * testMintEqualsBurnApplicationPercentage))

	// Account for rounding by ensuring all distributions sum to the total
	calculatedTotal := expectedDaoRewardAmount.Add(expectedProposerRewardAmount).Add(expectedSupplierRewardAmount).Add(expectedSourceOwnerRewardAmount).Add(expectedApplicationCostAmount)
	roundingDifference := totalSettlementAmount.Sub(calculatedTotal)

	// Give any rounding difference to the DAO (largest recipient)
	expectedDaoRewardAmount = expectedDaoRewardAmount.Add(roundingDifference)

	// Capture balances after settlement
	daoBalanceAfterSettlement := getBalance(t, ctx, keepers, daoRewardAddress)
	proposerBalanceAfterSettlement := getBalance(t, ctx, keepers, blockProposerAccountAddress)
	sourceOwnerBalanceAfterSettlement := getBalance(t, ctx, keepers, serviceSourceOwnerAddress)
	applicationBalanceAfterSettlement := getBalance(t, ctx, keepers, testApplicationAddress)

	// Verify DAO received expected reward distribution
	actualDaoRewardAmount := daoBalanceAfterSettlement.Amount.Sub(daoBalanceBeforeSettlement.Amount)
	require.Equal(t, expectedDaoRewardAmount, actualDaoRewardAmount,
		"DAO reward amount mismatch: expected %s, got %s", expectedDaoRewardAmount, actualDaoRewardAmount)

	// Verify proposer received expected reward distribution
	actualProposerRewardAmount := proposerBalanceAfterSettlement.Amount.Sub(proposerBalanceBeforeSettlement.Amount)
	require.Equal(t, expectedProposerRewardAmount, actualProposerRewardAmount,
		"Proposer reward amount mismatch: expected %s, got %s", expectedProposerRewardAmount, actualProposerRewardAmount)

	// Verify source owner received expected reward distribution
	actualSourceOwnerRewardAmount := sourceOwnerBalanceAfterSettlement.Amount.Sub(sourceOwnerBalanceBeforeSettlement.Amount)
	require.Equal(t, expectedSourceOwnerRewardAmount, actualSourceOwnerRewardAmount,
		"Source owner reward amount mismatch: expected %s, got %s", expectedSourceOwnerRewardAmount, actualSourceOwnerRewardAmount)

	// Verify application stake was reduced by expected cost (should be zero for MintEqualsBurn)
	actualApplicationCostAmount := applicationBalanceBeforeSettlement.Amount.Sub(applicationBalanceAfterSettlement.Amount)
	require.Equal(t, expectedApplicationCostAmount, actualApplicationCostAmount,
		"Application cost amount mismatch: expected %s, got %s", expectedApplicationCostAmount, actualApplicationCostAmount)

	// Verify supplier shareholders received expected reward distribution
	expectedSupplierShareholderRewardAmounts := tlm.GetShareAmountMap(supplierRevenueShareholders, expectedSupplierRewardAmount)
	for shareholderAddress, expectedShareholderRewardAmount := range expectedSupplierShareholderRewardAmounts {
		shareholderBalanceAfterSettlement := getBalance(t, ctx, keepers, shareholderAddress)
		shareholderBalanceBeforeSettlement := supplierShareholderBalancesBeforeSettlement[shareholderAddress]

		actualShareholderRewardAmount := shareholderBalanceAfterSettlement.Amount.Sub(shareholderBalanceBeforeSettlement.Amount)
		require.Equal(t, expectedShareholderRewardAmount, actualShareholderRewardAmount,
			"Supplier shareholder %s reward amount mismatch: expected %s, got %s",
			shareholderAddress, expectedShareholderRewardAmount, actualShareholderRewardAmount)
	}

	// Verify total reward distribution equals settlement amount
	totalDistributedAmount := actualDaoRewardAmount.Add(actualProposerRewardAmount).Add(expectedSupplierRewardAmount).Add(actualSourceOwnerRewardAmount).Add(actualApplicationCostAmount)
	require.Equal(t, totalSettlementAmount, totalDistributedAmount,
		"Total distributed amount mismatch: expected %s, got %s", totalSettlementAmount, totalDistributedAmount)
}
