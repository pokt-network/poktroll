package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgServer_StakeSupplier_SuccessfulCreateAndUpdate(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)

	// Prepare the stakeMsg
	stakeMsg := stakeSupplierForServicesMsg(supplierAddr, 100, "svcId")

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierAddr, foundSupplier.Address)
	require.Equal(t, int64(100), foundSupplier.Stake.Amount.Int64())
	require.Len(t, foundSupplier.Services, 1)
	require.Equal(t, "svcId", foundSupplier.Services[0].Service.Id)
	require.Len(t, foundSupplier.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", foundSupplier.Services[0].Endpoints[0].Url)

	// Prepare an updated supplier with a higher stake and a different URL for the service
	updateMsg := stakeSupplierForServicesMsg(supplierAddr, 200, "svcId2")

	// Update the staked supplier
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.NoError(t, err)

	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(200), foundSupplier.Stake.Amount.Int64())
	require.Len(t, foundSupplier.Services, 1)
	require.Equal(t, "svcId2", foundSupplier.Services[0].Service.Id)
	require.Len(t, foundSupplier.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", foundSupplier.Services[0].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailRestakingDueToInvalidServices(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	supplierAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg := stakeSupplierForServicesMsg(supplierAddr, 100, "svcId")

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the supplier stake message without any service endpoints
	updateStakeMsg := &types.MsgStakeSupplier{
		Address: supplierAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service:   &sharedtypes.Service{Id: "svcId"},
				Endpoints: []*sharedtypes.SupplierEndpoint{},
			},
		},
	}

	// Fail updating the supplier when the list of service endpoints is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the supplierFound still exists and is staked for svc1
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierAddr, supplierFound.Address)
	require.Len(t, supplierFound.Services, 1)
	require.Equal(t, "svcId", supplierFound.Services[0].Service.Id)
	require.Len(t, supplierFound.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", supplierFound.Services[0].Endpoints[0].Url)

	// Prepare the supplier stake message with an invalid service ID
	updateStakeMsg = &types.MsgStakeSupplier{
		Address: supplierAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1 INVALID ! & *"},
			},
		},
	}

	// Fail updating the supplier when the list of services is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the supplier still exists and is staked for svc1
	supplierFound, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierAddr, supplierFound.Address)
	require.Len(t, supplierFound.Services, 1)
	require.Equal(t, "svcId", supplierFound.Services[0].Service.Id)
	require.Len(t, supplierFound.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", supplierFound.Services[0].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailLoweringStake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Prepare the supplier
	supplierAddr := sample.AccAddress()
	stakeMsg := stakeSupplierForServicesMsg(supplierAddr, 100, "svcId")

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)

	// Prepare an updated supplier with a lower stake
	updateMsg := stakeSupplierForServicesMsg(supplierAddr, 50, "svcId")

	// Verify that it fails
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(100), supplierFound.Stake.Amount.Int64())
	require.Len(t, supplierFound.Services, 1)
}

func TestMsgServer_StakeSupplier_FailWithNonExistingService(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Prepare the supplier
	supplierAddr := sample.AccAddress()
	stakeMsg := stakeSupplierForServicesMsg(supplierAddr, 100, "newService")

	// Stake the supplier & verify that it fails because the service does not exist.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorIs(t, err, types.ErrSupplierServiceNotFound)
}

func TestMsgServer_StakeSupplier_ActiveSupplier(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Prepare the supplier
	supplierAddr := sample.AccAddress()
	stakeMsg := stakeSupplierForServicesMsg(supplierAddr, 100, "svcId")

	// Stake the supplier & verify that the supplier exists.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := supplierModuleKeepers.SharedKeeper.GetSessionEndHeight(ctx, currentHeight)

	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, 1, len(foundSupplier.ServicesActivationHeightsMap))

	// The supplier should have the service svcId activation height set to the
	// beginning of the next session.
	require.Equal(t, uint64(sessionEndHeight+1), foundSupplier.ServicesActivationHeightsMap["svcId"])

	// The supplier should be inactive for the service until the next session.
	require.False(t, foundSupplier.IsActive(uint64(currentHeight), "svcId"))
	require.False(t, foundSupplier.IsActive(uint64(sessionEndHeight), "svcId"))

	// The supplier should be active for the service in the next session.
	require.True(t, foundSupplier.IsActive(uint64(sessionEndHeight+1), "svcId"))

	// Set the chain height to the beginning of the next session.
	ctx = keepertest.SetBlockHeight(ctx, sessionEndHeight+1)

	// Prepare the supplier stake message with a different service
	updateMsg := stakeSupplierForServicesMsg(supplierAddr, 200, "svcId", "svcId2")

	// Update the staked supplier
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.NoError(t, err)

	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)

	// The supplier should reference both services.
	require.Equal(t, 2, len(foundSupplier.ServicesActivationHeightsMap))

	// svcId activation height should remain the same.
	require.Equal(t, uint64(sessionEndHeight+1), foundSupplier.ServicesActivationHeightsMap["svcId"])

	// svcId2 activation height should be the beginning of the next session.
	nextSessionEndHeight := supplierModuleKeepers.SharedKeeper.GetSessionEndHeight(ctx, sessionEndHeight+1)
	require.Equal(t, uint64(nextSessionEndHeight+1), foundSupplier.ServicesActivationHeightsMap["svcId2"])

	// The supplier should be active only for svcId until the end of the current session.
	require.True(t, foundSupplier.IsActive(uint64(nextSessionEndHeight), "svcId"))
	require.False(t, foundSupplier.IsActive(uint64(nextSessionEndHeight), "svcId2"))

	// The supplier should be active for both services in the next session.
	require.True(t, foundSupplier.IsActive(uint64(nextSessionEndHeight+1), "svcId"))
	require.True(t, foundSupplier.IsActive(uint64(nextSessionEndHeight+1), "svcId2"))
}

// stakeSupplierForServicesMsg prepares and returns a MsgStakeSupplier that stakes
// the given supplier address, stake amount, and service IDs.
func stakeSupplierForServicesMsg(
	supplierAddr string,
	amount int64,
	serviceIds ...string,
) *types.MsgStakeSupplier {
	services := make([]*sharedtypes.SupplierServiceConfig, 0, len(serviceIds))
	for _, serviceId := range serviceIds {
		services = append(services, &sharedtypes.SupplierServiceConfig{
			Service: &sharedtypes.Service{Id: serviceId},
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://localhost:8080",
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
		})
	}

	return &types.MsgStakeSupplier{
		Address:  supplierAddr,
		Stake:    &sdk.Coin{Denom: volatile.DenomuPOKT, Amount: math.NewInt(amount)},
		Services: services,
	}
}
