package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
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
	stakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, operatorAddr, foundSupplier.Address)
	require.Equal(t, int64(100), foundSupplier.Stake.Amount.Int64())
	require.Len(t, foundSupplier.Services, 1)
	require.Equal(t, "svcId", foundSupplier.Services[0].Service.Id)
	require.Len(t, foundSupplier.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", foundSupplier.Services[0].Endpoints[0].Url)

	// Prepare an updated supplier with a higher stake and a different URL for the service
	updateMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 200)
	updateMsg.Sender = operatorAddr
	updateMsg.Services[0].Endpoints[0].Url = "http://localhost:8082"
	updateMsg.Services[0].Service.Id = "svcId2"

	// Update the staked supplier
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.NoError(t, err)

	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(200), foundSupplier.Stake.Amount.Int64())
	require.Len(t, foundSupplier.Services, 1)
	require.Equal(t, "svcId2", foundSupplier.Services[0].Service.Id)
	require.Len(t, foundSupplier.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8082", foundSupplier.Services[0].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailRestakingDueToInvalidServices(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the supplier stake message without any service endpoints
	updateStakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 200)
	updateStakeMsg.Sender = operatorAddr
	updateStakeMsg.Services[0].Endpoints = []*sharedtypes.SupplierEndpoint{}

	// Fail updating the supplier when the list of service endpoints is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the supplierFound still exists and is staked for svc1
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, operatorAddr, supplierFound.Address)
	require.Len(t, supplierFound.Services, 1)
	require.Equal(t, "svcId", supplierFound.Services[0].Service.Id)
	require.Len(t, supplierFound.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", supplierFound.Services[0].Endpoints[0].Url)

	// Prepare the supplier stake message with an invalid service ID
	updateStakeMsg = prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 200)
	updateStakeMsg.Services[0].Service.Id = "svc1 INVALID ! & *"

	// Fail updating the supplier when the list of services is empty
	_, err = srv.StakeSupplier(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the supplier still exists and is staked for svc1
	supplierFound, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, operatorAddr, supplierFound.Address)
	require.Len(t, supplierFound.Services, 1)
	require.Equal(t, "svcId", supplierFound.Services[0].Service.Id)
	require.Len(t, supplierFound.Services[0].Endpoints, 1)
	require.Equal(t, "http://localhost:8080", supplierFound.Services[0].Endpoints[0].Url)
}

func TestMsgServer_StakeSupplier_FailLoweringStake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message
	stakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)

	// Stake the supplier & verify that the supplier exists
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)

	// Prepare an update supplier msg with a lower stake
	updateMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 50)
	updateMsg.Sender = operatorAddr

	// Verify that it fails
	_, err = srv.StakeSupplier(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the supplier stake is unchanged
	supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, operatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(100), supplierFound.Stake.Amount.Int64())
	require.Len(t, supplierFound.Services, 1)
}

func TestMsgServer_StakeSupplier_FailWithNonExistingService(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Prepare the supplier stake message with a non-existing service ID
	stakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)
	stakeMsg.Services[0].Service.Id = "newService"

	// Stake the supplier & verify that it fails because the service does not exist.
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorIs(t, err, types.ErrSupplierServiceNotFound)
}

func TestMsgServer_StakeSupplier_OperatorAuthorizations(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)

	// Generate an owner and operator address for the supplier
	ownerAddr := sample.AccAddress()
	operatorAddr := sample.AccAddress()

	// Try staking the supplier & verify that it fails because the operator is not
	// allowed to stake.
	stakeMsg := prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)
	stakeMsg.Sender = operatorAddr
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedUnauthorizedSupplierUpdate)

	// Prepare a supplier stake msg with the owner address as the sender and verify
	// that it succeeds.
	stakeMsg = prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 100)
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Update the supplier using the operator address as the sender and verify that it succeeds.
	stakeMsg = prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 200)
	stakeMsg.Sender = operatorAddr
	newUrl := "http://localhost:8081"
	stakeMsg.Services[0].Endpoints[0].Url = newUrl
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Check that the supplier was updated
	foundSupplier, supplierFound := k.GetSupplier(ctx, operatorAddr)
	require.True(t, supplierFound)
	require.Equal(t, newUrl, foundSupplier.Services[0].Endpoints[0].Url)

	// Try updating the supplier using the owner address as the sender and verify that it fails.
	stakeMsg = prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 300)
	newUrl = "http://localhost:8081"
	stakeMsg.Services[0].Endpoints[0].Url = newUrl
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedUnauthorizedSupplierUpdate)

	// Try updating the supplier's owner address and verify that it fails.
	stakeMsg = prepareValidMsgStakeSupplier(ownerAddr, operatorAddr, 300)
	stakeMsg.Sender = operatorAddr
	stakeMsg.OwnerAddress = sample.AccAddress()
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedUnauthorizedSupplierUpdate)
}

// prepareValidMsgStakeSupplier prepares a valid MsgStakeSupplier with the given
// owner, operator address and stake amount.
func prepareValidMsgStakeSupplier(ownerAddr, operatorAddr string, amount int64) *types.MsgStakeSupplier {
	return &types.MsgStakeSupplier{
		Sender:       ownerAddr,
		OwnerAddress: ownerAddr,
		Address:      operatorAddr,
		Stake:        &sdk.Coin{Denom: "upokt", Amount: math.NewInt(amount)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{
					Id: "svcId",
				},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     "http://localhost:8080",
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		},
	}
}
