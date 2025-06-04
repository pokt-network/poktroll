package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_0_1_16_PlanName = "v0.1.16"
)

// Upgrade_0_1_16 handles the upgrade to release `v0.1.16`.
// - Normalize Morse accounts recovery allowlist addresses (to uppercase).
// - Normalize Morse source address when handling Morse account recovery message.
var Upgrade_0_1_16 = Upgrade{
	PlanName: Upgrade_0_1_16_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.15..v0.1.16
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.15..v0.1.16

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()

			// Adds new authz that were previously incorrect. See #1425
			// These can be validated like so:
			// pocketd query authz grants-by-granter <addr> --network=<network> -o json --grpc-insecure=false
			// 	| jq '.grants[]|select(.authorization.value.msg == "/pocket.migration.MsgRecoverMorseAccount")'
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParams",
				"/pocket.service.MsgRecoverMorseAccount",
			}
			if err := applyNewAuthorizations(ctx, keepers, logger, grantAuthorizationMessages); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
