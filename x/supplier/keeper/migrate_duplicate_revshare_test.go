package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestDeduplicateSupplierRevShareAddresses_NoDuplicates(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	keeper := supplierModuleKeepers.Keeper

	// Create a supplier with unique rev share addresses
	addr1 := sample.AccAddressBech32()
	addr2 := sample.AccAddressBech32()
	supplier := sharedtypes.Supplier{
		OwnerAddress:    addr1,
		OperatorAddress: addr1,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(1000000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: "svc1",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{Url: "http://localhost:8080", RpcType: sharedtypes.RPCType_JSON_RPC},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{Address: addr1, RevSharePercentage: 30},
					{Address: addr2, RevSharePercentage: 70},
				},
			},
		},
	}
	supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplier.OperatorAddress, supplier.Services, 1, sharedtypes.NoDeactivationHeight,
	)
	keeper.SetAndIndexDehydratedSupplier(ctx, supplier)

	count, err := keeper.DeduplicateSupplierRevShareAddresses(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count, "no suppliers should be modified when there are no duplicates")
}

func TestDeduplicateSupplierRevShareAddresses_WithDuplicates(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	keeper := supplierModuleKeepers.Keeper

	// Simulate the mainnet case: [{operator:15}, {owner:15}, {owner:70}]
	operatorAddr := sample.AccAddressBech32()
	ownerAddr := sample.AccAddressBech32()
	supplier := sharedtypes.Supplier{
		OwnerAddress:    ownerAddr,
		OperatorAddress: operatorAddr,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(60100000000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: "eth",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{Url: "http://localhost:8080", RpcType: sharedtypes.RPCType_JSON_RPC},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{Address: operatorAddr, RevSharePercentage: 15},
					{Address: ownerAddr, RevSharePercentage: 15},
					{Address: ownerAddr, RevSharePercentage: 70},
				},
			},
			{
				ServiceId: "gnosis",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{Url: "http://localhost:8081", RpcType: sharedtypes.RPCType_JSON_RPC},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{Address: ownerAddr, RevSharePercentage: 15},
					{Address: ownerAddr, RevSharePercentage: 85},
				},
			},
		},
	}
	supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		supplier.OperatorAddress, supplier.Services, 1, sharedtypes.NoDeactivationHeight,
	)
	keeper.SetAndIndexDehydratedSupplier(ctx, supplier)

	count, err := keeper.DeduplicateSupplierRevShareAddresses(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "one supplier should be modified")

	// Verify the supplier's config was fixed
	fixedSupplier, found := keeper.GetSupplier(ctx, operatorAddr)
	require.True(t, found)

	// Check each service's rev share was deduplicated
	for _, configUpdate := range fixedSupplier.ServiceConfigHistory {
		svc := configUpdate.Service
		// Verify no duplicate addresses remain
		seen := make(map[string]struct{})
		for _, rs := range svc.RevShare {
			_, exists := seen[rs.Address]
			require.False(t, exists, "duplicate address found after dedup: %s in service %s", rs.Address, svc.ServiceId)
			seen[rs.Address] = struct{}{}
		}

		// Verify percentages sum to 100
		var sum uint64
		for _, rs := range svc.RevShare {
			sum += rs.RevSharePercentage
		}
		require.Equal(t, uint64(100), sum, "rev share percentages should sum to 100 for service %s", svc.ServiceId)

		// Verify specific merged values
		switch svc.ServiceId {
		case "eth":
			require.Len(t, svc.RevShare, 2, "eth should have 2 unique addresses")
			for _, rs := range svc.RevShare {
				switch rs.Address {
				case operatorAddr:
					require.Equal(t, uint64(15), rs.RevSharePercentage)
				case ownerAddr:
					require.Equal(t, uint64(85), rs.RevSharePercentage) // 15 + 70
				}
			}
		case "gnosis":
			require.Len(t, svc.RevShare, 1, "gnosis should have 1 unique address")
			require.Equal(t, ownerAddr, svc.RevShare[0].Address)
			require.Equal(t, uint64(100), svc.RevShare[0].RevSharePercentage) // 15 + 85
		}
	}
}

func TestDeduplicateSupplierRevShareAddresses_MixedSuppliers(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	keeper := supplierModuleKeepers.Keeper

	// Create a clean supplier (no duplicates)
	cleanAddr := sample.AccAddressBech32()
	cleanSupplier := sharedtypes.Supplier{
		OwnerAddress:    cleanAddr,
		OperatorAddress: cleanAddr,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(1000000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: "svc1",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{Url: "http://localhost:8080", RpcType: sharedtypes.RPCType_JSON_RPC},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{Address: cleanAddr, RevSharePercentage: 100},
				},
			},
		},
	}
	cleanSupplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		cleanSupplier.OperatorAddress, cleanSupplier.Services, 1, sharedtypes.NoDeactivationHeight,
	)
	keeper.SetAndIndexDehydratedSupplier(ctx, cleanSupplier)

	// Create a supplier with duplicates
	dupAddr := sample.AccAddressBech32()
	dupSupplier := sharedtypes.Supplier{
		OwnerAddress:    dupAddr,
		OperatorAddress: dupAddr,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(1000000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: "svc2",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{Url: "http://localhost:8081", RpcType: sharedtypes.RPCType_JSON_RPC},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{Address: dupAddr, RevSharePercentage: 40},
					{Address: dupAddr, RevSharePercentage: 60},
				},
			},
		},
	}
	dupSupplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		dupSupplier.OperatorAddress, dupSupplier.Services, 1, sharedtypes.NoDeactivationHeight,
	)
	keeper.SetAndIndexDehydratedSupplier(ctx, dupSupplier)

	count, err := keeper.DeduplicateSupplierRevShareAddresses(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "only the supplier with duplicates should be modified")
}
