package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgServer_StakeSupplier_SuccessfulCreateAndUpdate(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.False(t, isSupplierFound)

	// Prepare the stakeMsg
	stakeMsg, expectedSupplier := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId")

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the EventSupplierStaked event is emitted.
	events := cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")

	sessionEndHeight := supplierModuleKeepers.SharedKeeper.GetSessionEndHeight(ctx, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	expectedEvent, err := cosmostypes.TypedEventToEvent(
		&suppliertypes.EventSupplierStaked{
			Supplier:         expectedSupplier,
			SessionEndHeight: sessionEndHeight,
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, expectedEvent, events[0])

	// Verify that the supplier:
	// - Is staked
	// - Has no active services yet
	// - Has a service update scheduled for the next session
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, operatorAddr, foundSupplier.OperatorAddress)
	require.Equal(t, int64(1000000), foundSupplier.Stake.Amount.Int64())
	require.Len(t, foundSupplier.Services, 0)

	latestConfigUpdate := getLatestSupplierServiceConfigUpdate(t, foundSupplier)
	require.Len(t, foundSupplier.ServiceConfigHistory, 1)
	require.Equal(t, "svcId", latestConfigUpdate.Services[0].ServiceId)

	// Reset the events, as if a new block were created.
	ctx, _ = testevents.ResetEventManager(ctx)

	// Activate the supplier's services
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	numSuppliersWithServicesActivation, err := supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Verify that the supplier has its services activated.
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Len(t, foundSupplier.Services, 1)
	require.Equal(t, "svcId", foundSupplier.Services[0].ServiceId)
	require.Len(t, foundSupplier.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", foundSupplier.Services[0].Endpoints[0].Url)
	// Assert that the supplier's account balance was reduced by the staking fee
	supplierStakingFee := supplierModuleKeepers.Keeper.GetParams(ctx).StakingFee
	balanceDecrease := supplierStakingFee.Amount.Int64() + foundSupplier.Stake.Amount.Int64()
	// SupplierBalanceMap reflects the relative changes to the supplier's balance
	// (i.e. it starts from 0 and can go below it).
	// It is not using coins that enforce non-negativity of the balance nor account
	// funding and lookups.
	require.Equal(t, -balanceDecrease, supplierModuleKeepers.SupplierBalanceMap[ownerAddr])

	// Prepare an updated supplier with the same stake and an additional service.
	updateMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId", "svcId2")
	updateMsg.Services[0].Endpoints[0].Url = "http://localhost:8080"
	updateMsg.Services[1].Endpoints[0].Url = "http://localhost:8082"

	// Update the staked supplier
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.NoError(t, err)

	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(1000000), foundSupplier.Stake.Amount.Int64())
	// Ensure that the supplier new service is not active yet.
	require.Len(t, foundSupplier.Services, 1)
	require.Len(t, foundSupplier.ServiceConfigHistory, 2)

	// Check that the supplier config update contains the new service the supplier
	// has staked for.
	latestConfigUpdate = getLatestSupplierServiceConfigUpdate(t, foundSupplier)
	require.Equal(t, "svcId2", latestConfigUpdate.Services[1].ServiceId)
	require.Len(t, latestConfigUpdate.Services[1].Endpoints, 1)
	require.Equal(t, "http://localhost:8082", latestConfigUpdate.Services[1].Endpoints[0].Url)

	// Activate the latest supplier's services update.
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	numSuppliersWithServicesActivation, err = supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Confirm that the latest service update is now active and reflected in the
	// supplier.Services field.
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Len(t, foundSupplier.Services, 2)
	require.Equal(t, "svcId2", foundSupplier.Services[1].ServiceId)
	require.Len(t, foundSupplier.Services[1].Endpoints, 1)
	require.Equal(t, "http://localhost:8082", foundSupplier.Services[1].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailRestakingDueToInvalidServices(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId")

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the supplier stake message without any service endpoints
	updateStakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 200, "svcId")
	updateStakeMsg.Services[0].Endpoints = []*sharedtypes.SupplierEndpoint{}

	// Fail updating the supplier when the list of service endpoints is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Activate the supplier's services
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	numSuppliersWithServicesActivation, err := supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Verify the supplierFound still exists and is staked for svc1
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierFound.Services[0].ServiceId, "svcId")

	// Prepare the supplier stake message with an invalid service ID
	updateStakeMsg, _ = newSupplierStakeMsg(ownerAddr, operatorAddr, 200, "svcId")
	updateStakeMsg.Services[0].ServiceId = "svc1 INVALID ! & *"

	// Fail updating the supplier when the list of services is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the supplier still exists and is staked for svc1
	supplierFound, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, operatorAddr, supplierFound.OperatorAddress)
	require.Len(t, supplierFound.Services, 1)
	require.Equal(t, "svcId", supplierFound.Services[0].ServiceId)
	require.Len(t, supplierFound.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", supplierFound.Services[0].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailLoweringStakeBelowMinStake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	minStake := supplierModuleKeepers.Keeper.GetParams(ctx).MinStake.Amount.Int64()

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake, "svcId")

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)

	// Prepare an update supplier msg with a lower stake
	updateMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake-1, "svcId")
	updateMsg.Signer = operatorAddr

	// Verify that it fails
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(1000000), supplierFound.Stake.Amount.Int64())
}

func TestMsgServer_StakeSupplier_SuccessLoweringStakeAboveMinStake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	minStake := supplierModuleKeepers.Keeper.GetParams(ctx).MinStake.Amount.Int64()

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake, "svcId")

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)

	// Prepare an updated supplier msg with a lower stake which is below the minimum staking fee.
	newStake := minStake - 1
	updateMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, newStake, "svcId")
	updateMsg.Signer = operatorAddr

	// Verify that it fails
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, minStake, supplierFound.Stake.Amount.Int64())
}

func TestMsgServer_StakeSupplier_SuccessIncreasingStake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	minStake := supplierModuleKeepers.Keeper.GetParams(ctx).MinStake.Amount.Int64()

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake, "svcId")

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)

	// Prepare an update supplier msg with a higher stake.
	newStake := minStake + 1
	updateMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, newStake, "svcId")
	updateMsg.Signer = operatorAddr

	// Verify that succeeds
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.NoError(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, newStake, supplierFound.Stake.Amount.Int64())
}

func TestMsgServer_StakeSupplier_FailWithNonExistingService(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message with a non-existing service ID
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "newService")

	// Stake the supplier & verify that it fails because the service does not exist.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierServiceNotFound.Wrapf(
		"service %q does not exist", "newService",
	).Error())

	// Verify that no EventSupplierStaked events were emitted.
	events := cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	require.Empty(t, events)

	// Verify that the supplier does not exist
	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_StakeSupplier_OperatorAuthorizations(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Stake using the operator address as the signer and verify that it succeeds.
	stakeMsg, expectedSupplier := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId")
	setStakeMsgSigner(stakeMsg, operatorAddr)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	supplier, isSupplierFound := k.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.EqualValues(t, expectedSupplier, &supplier)

	// Activate the supplier's services
	ctx = setBlockHeightToNextSessionStart(ctx, k.SharedKeeper)
	numSuppliersWithServicesActivation, err := k.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Update the supplier using the operator address as the signer and verify that it succeeds.
	stakeMsgUpdateUrl, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 2000000, "svcId")
	operatorUpdatedServiceUrl := "http://localhost:8081"
	stakeMsgUpdateUrl.Services[0].Endpoints[0].Url = operatorUpdatedServiceUrl
	setStakeMsgSigner(stakeMsgUpdateUrl, operatorAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateUrl)
	require.NoError(t, err)

	// Activate the supplier's services update
	ctx = setBlockHeightToNextSessionStart(ctx, k.SharedKeeper)
	numSuppliersWithServicesActivation, err = k.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Check that the supplier was updated
	foundSupplier, supplierFound := k.GetSupplier(ctx, operatorAddr)
	require.True(t, supplierFound)
	require.Equal(t, operatorUpdatedServiceUrl, foundSupplier.Services[0].Endpoints[0].Url)

	// Update the supplier URL by using the owner address as the singer and verify that it succeeds.
	ownerUpdaterServiceUrl := "http://localhost:8082"
	stakeMsgUpdateUrl.Services[0].Endpoints[0].Url = ownerUpdaterServiceUrl
	stakeMsgUpdateUrl.Stake.Amount = math.NewInt(3000000)
	setStakeMsgSigner(stakeMsgUpdateUrl, ownerAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateUrl)
	require.NoError(t, err)

	// Activate the supplier's services update
	ctx = setBlockHeightToNextSessionStart(ctx, k.SharedKeeper)
	numSuppliersWithServicesActivation, err = k.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numSuppliersWithServicesActivation)

	// Check that the supplier was updated
	foundSupplier, supplierFound = k.GetSupplier(ctx, operatorAddr)
	require.True(t, supplierFound)
	require.Equal(t, ownerUpdaterServiceUrl, foundSupplier.Services[0].Endpoints[0].Url)
	require.NotEqual(t, operatorUpdatedServiceUrl, foundSupplier.Services[0].Endpoints[0].Url)

	// Try updating the supplier's operator address using the old operator as a signer
	// will create a new supplier.
	stakeMsgUpdateOperator, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 3000000, "svcId")
	newOperatorAddress := sample.AccAddress()
	stakeMsgUpdateOperator.OperatorAddress = newOperatorAddress
	setStakeMsgSigner(stakeMsgUpdateOperator, operatorAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateOperator)
	require.NoError(t, err)

	// Check that the old supplier still exists.
	oldSupplier, oldSupplierFound := k.GetSupplier(ctx, operatorAddr)
	require.True(t, oldSupplierFound)
	// Check that a supplier with the new operator address exists.
	newSupplier, newSupplierFound := k.GetSupplier(ctx, newOperatorAddress)
	require.True(t, newSupplierFound)
	// Check that the old supplier is different from the new supplier.
	require.NotEqual(t, oldSupplier.OperatorAddress, newSupplier.OperatorAddress)

	// Trying to update the supplier's operator address using the owner as a signer
	// will create a new supplier.
	newOperatorAddress = sample.AccAddress()
	stakeMsgUpdateOperator.OperatorAddress = newOperatorAddress
	stakeMsgUpdateOperator.Stake.Amount = math.NewInt(4000000)
	setStakeMsgSigner(stakeMsgUpdateOperator, ownerAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateOperator)
	require.NoError(t, err)

	// Check that the old supplier still exists.
	oldSupplier, oldSupplierFound = k.GetSupplier(ctx, operatorAddr)
	require.True(t, oldSupplierFound)
	// Check that a supplier with the new operator address exists.
	newSupplier, newSupplierFound = k.GetSupplier(ctx, newOperatorAddress)
	require.True(t, newSupplierFound)
	// Check that the old supplier is different from the new supplier.
	require.NotEqual(t, oldSupplier.OperatorAddress, newSupplier.OperatorAddress)

	// Try updating the supplier's owner address using the operator as a signer
	// and verify that it fails.
	newOwnerAddress := sample.AccAddress()
	stakeMsgUpdateOwner, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 5000000, "svcId")
	stakeMsgUpdateOwner.OwnerAddress = newOwnerAddress
	setStakeMsgSigner(stakeMsgUpdateOwner, operatorAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateOwner)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
		"signer %q is not allowed to update the owner address %q",
		operatorAddr, ownerAddr,
	).Error())

	// Update the supplier's owner address using the owner as a signer and verify that it succeeds.
	setStakeMsgSigner(stakeMsgUpdateOwner, ownerAddr)
	_, err = srv.StakeSupplier(ctx, stakeMsgUpdateOwner)
	require.NoError(t, err)

	// Check that the supplier was updated.
	foundSupplier, supplierFound = k.GetSupplier(ctx, operatorAddr)
	require.True(t, supplierFound)
	require.Equal(t, newOwnerAddress, foundSupplier.OwnerAddress)
}

func TestMsgServer_StakeSupplier_ActiveSupplier(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier
	stakeMsg, expectedSupplier := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId")

	// Stake the supplier & verify that the supplier exists.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := supplierModuleKeepers.SharedKeeper.GetSessionEndHeight(sdkCtx, currentHeight)

	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(sdkCtx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Len(t, foundSupplier.ServiceConfigHistory, 1)
	require.EqualValues(t, expectedSupplier, &foundSupplier)

	latestServiceUpdate := getLatestSupplierServiceConfigUpdate(t, foundSupplier)
	// The supplier should have the service svcId activation height set to the
	// beginning of the next session.
	require.Equal(t, uint64(sessionEndHeight+1), latestServiceUpdate.EffectiveBlockHeight)

	// The supplier should be inactive for the service until the next session.
	require.False(t, foundSupplier.IsActive(uint64(currentHeight), "svcId"))
	require.False(t, foundSupplier.IsActive(uint64(sessionEndHeight), "svcId"))

	// The supplier should be active for the service in the next session.
	require.True(t, foundSupplier.IsActive(uint64(sessionEndHeight+1), "svcId"))

	ctx = setBlockHeightToNextSessionStart(sdkCtx, supplierModuleKeepers.SharedKeeper)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	currentHeight = sdkCtx.BlockHeight()

	// Prepare the supplier stake message with a different service
	updateMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 2000000, "svcId", "svcId2")
	updateMsg.Signer = operatorAddr

	// Update the staked supplier
	_, err = srv.StakeSupplier(sdkCtx, updateMsg)
	require.NoError(t, err)

	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(sdkCtx, operatorAddr)
	require.True(t, isSupplierFound)

	// The supplier should reference both services.
	require.Equal(t, 2, len(foundSupplier.ServiceConfigHistory))

	latestServiceUpdate = getLatestSupplierServiceConfigUpdate(t, foundSupplier)

	// The latest service update should contain both services
	require.Equal(t, 2, len(latestServiceUpdate.Services))
	require.Equal(t, "svcId", latestServiceUpdate.Services[0].ServiceId)
	require.Equal(t, "svcId2", latestServiceUpdate.Services[1].ServiceId)

	// Activation height should be the beginning of the next session.
	sessionEndHeight = supplierModuleKeepers.SharedKeeper.GetSessionEndHeight(sdkCtx, currentHeight)
	nextSessionStartHeight := sessionEndHeight + 1
	require.Equal(t, uint64(nextSessionStartHeight), latestServiceUpdate.EffectiveBlockHeight)

	// The supplier should be active only for svcId until the end of the current session.
	require.True(t, foundSupplier.IsActive(uint64(sessionEndHeight), "svcId"))
	require.False(t, foundSupplier.IsActive(uint64(sessionEndHeight), "svcId2"))

	// The supplier should be active for both svcId and svcId2 in the next session.
	require.True(t, foundSupplier.IsActive(uint64(nextSessionStartHeight), "svcId"))
	require.True(t, foundSupplier.IsActive(uint64(nextSessionStartHeight), "svcId2"))
}

func TestMsgServer_StakeSupplier_FailBelowMinStake(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)

	addr := sample.AccAddress()
	supplierStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
	minStake := supplierStake.AddAmount(math.NewInt(1))
	expectedErr := suppliertypes.ErrSupplierInvalidStake.Wrapf("supplier with owner %q must stake at least %s", addr, minStake)

	// Set the minimum stake to be greater than the supplier stake.
	params := k.Keeper.GetParams(ctx)
	params.MinStake = &minStake
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Prepare the supplier stake message.
	stakeMsg, _ := newSupplierStakeMsg(addr, addr, 100, "svcId")

	// Attempt to stake the supplier & verify that the supplier does NOT exist.
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorContains(t, err, expectedErr.Error())
	_, isSupplierFound := k.GetSupplier(ctx, addr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_StakeSupplier_UpStakeFromBelowMinStake(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)

	addr := sample.AccAddress()
	supplierParams := k.Keeper.GetParams(ctx)
	minStake := supplierParams.GetMinStake()
	belowMinStake := minStake.AddAmount(math.NewInt(-1))
	aboveMinStake := minStake.AddAmount(math.NewInt(1))

	stakeMsg, expectedSupplier := newSupplierStakeMsg(addr, addr, aboveMinStake.Amount.Int64(), "svcId")

	// Stake (via keeper methods) a supplier with stake below min stake.
	initialSupplier := sharedtypes.Supplier{
		OwnerAddress:    addr,
		OperatorAddress: addr,
		Stake:           &belowMinStake,
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
			{
				Services:             stakeMsg.GetServices(),
				EffectiveBlockHeight: 11,
			},
		},
	}
	k.SetSupplier(ctx, initialSupplier)

	// Attempt to upstake the supplier with stake above min stake.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert supplier is staked for above min stake.
	supplier, isSupplierFound := k.GetSupplier(ctx, addr)
	require.True(t, isSupplierFound)
	require.EqualValues(t, expectedSupplier, &supplier)
}

// newSupplierStakeMsg prepares and returns a MsgStakeSupplier that stakes
// the given supplier operator address, stake amount, and service IDs.
func newSupplierStakeMsg(
	ownerAddr, operatorAddr string,
	stakeAmount int64,
	serviceIds ...string,
) (stakeMsg *suppliertypes.MsgStakeSupplier, expectedSupplier *sharedtypes.Supplier) {
	services := make([]*sharedtypes.SupplierServiceConfig, 0, len(serviceIds))
	for _, serviceId := range serviceIds {
		services = append(services, &sharedtypes.SupplierServiceConfig{
			ServiceId: serviceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://localhost:8080",
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: nil,
				},
			},
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            ownerAddr,
					RevSharePercentage: 100,
				},
			},
		})
	}

	initialStake := cosmostypes.NewCoin("upokt", math.NewInt(stakeAmount))

	msg := &suppliertypes.MsgStakeSupplier{
		Signer:          ownerAddr,
		OwnerAddress:    ownerAddr,
		OperatorAddress: operatorAddr,
		Stake:           &initialStake,
		Services:        services,
	}

	supplier := &sharedtypes.Supplier{
		OwnerAddress:    ownerAddr,
		OperatorAddress: operatorAddr,
		Stake:           &initialStake,
		Services:        make([]*sharedtypes.SupplierServiceConfig, 0),
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
			{
				Services:             services,
				EffectiveBlockHeight: 11,
			},
		},
	}

	return msg, supplier
}

// setStakeMsgSigner sets the signer of the given MsgStakeSupplier to the given address
func setStakeMsgSigner(
	msg *suppliertypes.MsgStakeSupplier,
	signer string,
) *suppliertypes.MsgStakeSupplier {
	msg.Signer = signer
	return msg
}

// getLatestSupplierServiceConfigUpdate returns the latest service config update.
func getLatestSupplierServiceConfigUpdate(t *testing.T, supplier sharedtypes.Supplier) *sharedtypes.ServiceConfigUpdate {
	require.Greater(t, len(supplier.ServiceConfigHistory), 0)
	latestUpdateIdx := len(supplier.ServiceConfigHistory) - 1
	return supplier.ServiceConfigHistory[latestUpdateIdx]
}

// setBlockHeightToNextSessionStart sets the block height to the next session start height.
func setBlockHeightToNextSessionStart(
	ctx context.Context,
	sharedKeeper suppliertypes.SharedKeeper,
) context.Context {
	sharedParams := sharedKeeper.GetParams(ctx)
	currentHeight := cosmostypes.UnwrapSDKContext(ctx).BlockHeight()
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)
	return keepertest.SetBlockHeight(ctx, nextSessionStartHeight)
}
