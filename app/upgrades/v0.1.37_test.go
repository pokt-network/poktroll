package upgrades_test

import (
	"strings"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/app/upgrades"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TestUpgrade_0_1_37_PlanMetadata is a static sanity check on the v0.1.37 upgrade
// descriptor. A misnamed plan or a nil handler halts the chain at the upgrade height
// rather than upgrading it.
func TestUpgrade_0_1_37_PlanMetadata(t *testing.T) {
	require.Equal(t, "v0.1.37", upgrades.Upgrade_0_1_37.PlanName)
	require.Equal(t, "v0.1.37", upgrades.Upgrade_0_1_37_PlanName)
	require.Equal(t, storetypes.StoreUpgrades{}, upgrades.Upgrade_0_1_37.StoreUpgrades,
		"v0.1.37 creates no new KVStores")
	require.NotNil(t, upgrades.Upgrade_0_1_37.CreateUpgradeHandler)
}

// TestUpgrade_0_1_37_Handler runs the real v0.1.37 handler. Unlike earlier upgrade
// tests it does not replicate the handler's steps inline: the handler only uses the
// supplier, service, application and gateway keepers, all wired onto the supplier
// test keeper's store (their key prefixes are disjoint).
func TestUpgrade_0_1_37_Handler(t *testing.T) {
	k, ctx := testkeeper.SupplierKeeper(t)
	stake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, suppliertypes.DefaultMinStake.Amount.Int64())

	// Supplier stored under a non-canonical (uppercase) operator address.
	aliasOwner, aliasOperator := sample.AccAddressBech32(), strings.ToUpper(sample.AccAddressBech32())
	k.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{OwnerAddress: aliasOwner, OperatorAddress: aliasOperator, Stake: &stake})

	// Canonical supplier whose owner address was stored non-canonical.
	owner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	k.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{OwnerAddress: strings.ToUpper(owner), OperatorAddress: operator, Stake: &stake})

	// Service whose owner address was stored non-canonical.
	serviceOwner := sample.AccAddressBech32()
	k.ServiceKeeper.SetService(ctx, sharedtypes.Service{Id: "svc37", Name: "svc37", ComputeUnitsPerRelay: 1, OwnerAddress: strings.ToUpper(serviceOwner)})

	// Application and gateway stored under non-canonical (uppercase) addresses. Their
	// migrations only use the shared keeper, so the other dependencies are nil.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	sharedKeeper := k.SharedKeeper.(sharedkeeper.Keeper)
	storeService := runtime.NewKVStoreService(k.StoreKey.(*storetypes.KVStoreKey))
	appKeeper := appkeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, nil, nil, sharedKeeper)
	gatewayKeeper := gatewaykeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, sharedKeeper)
	aliasApp := strings.ToUpper(sample.AccAddressBech32())
	appKeeper.SetApplication(ctx, apptypes.Application{Address: aliasApp, Stake: &stake, ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svcId"}}})
	aliasGateway := strings.ToUpper(sample.AccAddressBech32())
	gatewayKeeper.SetGateway(ctx, gatewaytypes.Gateway{Address: aliasGateway, Stake: &stake})

	handler := upgrades.Upgrade_0_1_37.CreateUpgradeHandler(nil, &keepers.Keepers{
		SupplierKeeper:    *k.Keeper,
		ServiceKeeper:     k.ServiceKeeper,
		ApplicationKeeper: appKeeper,
		GatewayKeeper:     gatewayKeeper,
	}, nil)
	vm := module.VersionMap{suppliertypes.ModuleName: 1}
	gotVM, err := handler(ctx, upgradetypes.Plan{Name: upgrades.Upgrade_0_1_37_PlanName}, vm)
	require.NoError(t, err)
	require.Equal(t, vm, gotVM)

	_, found := k.GetDehydratedSupplier(ctx, aliasOperator)
	require.False(t, found, "non-canonical supplier must be removed")
	require.Zero(t, k.SupplierBalanceMap[aliasOwner], "removed supplier must not be refunded")

	supplier, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	require.Equal(t, owner, supplier.OwnerAddress)

	service, found := k.ServiceKeeper.GetService(ctx, "svc37")
	require.True(t, found)
	require.Equal(t, serviceOwner, service.OwnerAddress)

	app, found := appKeeper.GetApplication(ctx, aliasApp)
	require.True(t, found)
	require.True(t, app.IsUnbonding(), "non-canonical application must begin unbonding")
	gateway, found := gatewayKeeper.GetGateway(ctx, aliasGateway)
	require.True(t, found)
	require.True(t, gateway.IsUnbonding(), "non-canonical gateway must begin unbonding")
}
