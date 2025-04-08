---
title: Handling Protobufs Upgrades
description: How to handle protobuf upgrades in a backwards compatible way
sidebar_position: 3
---

# Handling Protobufs Upgrades

When making changes to protobuf definitions that require backwards compatibility during upgrades,
you may need to work with previous versions of protobuf definitions.

This guide explains how to handle such scenarios effectively using the `.deprecated.proto` convention.

## Overview

In blockchain upgrades, sometimes we need to change data structures while ensuring
smooth migration from the old structure to the new one. The approach involves:

1. Preserving the previous version of the protobuf definition using the `.deprecated.proto` file convention
2. Implementing migration logic in the upgrade handler to convert data from the previous format to the new one

## Step-by-Step Process

### 1. Preserve the Previous Protobuf Definition

When you need to change a protobuf definition but maintain compatibility for upgrades:

1. Create a new file with `.deprecated.proto` suffix containing the previous definition:

```protobuf
// Example: pocket/shared/supplier.deprecated.proto
syntax = "proto3";
package pocket.shared;

option go_package = "github.com/pokt-network/poktroll/x/shared/types";
option (gogoproto.stable_marshaler_all) = true;

// Include the original imports
import "cosmos_proto/cosmos.proto";
import "cosmos/base/v1beta1/coin.proto";
import "gogoproto/gogo.proto";
import "pocket/shared/service.proto";

// Previous proto definition (pre-update)
message SupplierDeprecated {
  // Unchanged fields from the previous definition
  string owner_address = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator_address = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  cosmos.base.v1beta1.Coin stake = 3;
  repeated SupplierServiceConfig services = 4;
  uint64 unstake_session_end_height = 5;

  // The upgrade will change this map to a ServiceConfigUpdate repeated field
  map<string, uint64> services_activation_heights_map = 6;
}
```

### 2. Add Keeper Methods for the Previous Type

Create methods in your keeper to handle the previous types:

```go
// Example in x/supplier/keeper/supplier.go

// GetAllSuppliersDeprecated returns all suppliers using the previous format
// TODO_NEXT_RELEASE: Remove this method prior to the next release
func (k Keeper) GetAllSuppliersDeprecated(ctx context.Context) (suppliers []sharedtypes.SupplierDeprecated) {
    storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
    store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
    iterator := storetypes.KVStorePrefixIterator(store, []byte{})

    defer iterator.Close()

    for ; iterator.Valid(); iterator.Next() {
        var supplier sharedtypes.SupplierDeprecated
        k.cdc.MustUnmarshal(iterator.Value(), &supplier)
        suppliers = append(suppliers, supplier)
    }

    return
}
```

### 3. Implement Migration Logic in the Upgrade Handler

In your upgrade handler, implement the logic to migrate from the previous structure to the new one:

```go
// Example in app/upgrades/v0.0.14.go
CreateUpgradeHandler: func(
  mm *module.Manager,
  keepers *keepers.Keepers,
  configurator module.Configurator,
) upgradetypes.UpgradeHandler {
    return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
        logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
        logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_0_14_PlanName)

        supplierKeeper := keepers.SupplierKeeper

        // Get all suppliers using the deprecated supplier keeper method
        suppliers := supplierKeeper.GetAllSuppliersDeprecated(ctx)

        for _, supplierDeprecated := range suppliers {
            // Convert previous format to new format
            supplier := sharedtypes.Supplier{
                OperatorAddress:         supplierDeprecated.OperatorAddress,
                Services:                supplierDeprecated.Services,
                OwnerAddress:            supplierDeprecated.OwnerAddress,
                Stake:                   supplierDeprecated.Stake,
                UnstakeSessionEndHeight: supplierDeprecated.UnstakeSessionEndHeight,
                // Add new fields or transform data as needed
                ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
                    {
                        Services:             supplierDeprecated.Services,
                        EffectiveBlockHeight: 1,
                    },
                },
            }

            // Update the supplier with the migrated data
            supplierKeeper.SetSupplier(ctx, supplier)

            logger.Info(
                "Successfully migrated supplier data",
                "supplier_address", supplier.OperatorAddress,
            )
        }

        // Continue with other migrations
        return mm.RunMigrations(ctx, configurator, vm)
    }
},
```

### 4. Clean Up After Upgrades

After the upgrade has been successfully deployed and the network has migrated:

1. Add TODOs to mark previous version code for removal in the next release:
```go
// TODO_NEXT_RELEASE: Remove this and other deprecated methods prior to v0.0.15 release
```

2. Remove the previous definitions and methods in the subsequent release once they're no longer needed.

## Best Practices

1. **File Naming Convention**: Use the `.deprecated.proto` suffix in filenames to
   clearly indicate previous versions of definitions, maintaining the established convention.

2. **Documentation**: Add comments explaining why the previous version exists and when it can be removed.

3. **Backwards Compatibility**: Ensure your migration logic handles all edge cases when converting between formats.

4. **Consistency**: Ensure your migration logic maintains blockchain state consistency by:
    1. Preserving all existing data during migration
    2. Properly initializing any new fields or structures
    3. Updating all references between old and new data structures
    4. Making the migration process idempotent (can be run multiple times safely)
    5. Fully populating the new data structure with all required information

5. **Testing**: Thoroughly test your upgrade handler with both the previous and new data formats.

6. **Cleanup**: Plan for the removal of previous version code in the next release after the upgrade.

## Example Use Cases

- Adding new fields to an existing structure
- Changing field types
- Restructuring nested objects
- Splitting or combining structures

By following this approach, you can make significant changes to your data structures
while ensuring a smooth upgrade process for the blockchain network.