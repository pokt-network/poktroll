package upgrades

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// defaultUpgradeHandler should be used for upgrades that only update the `ConsensusVersion`.
// If an upgrade involves state changes, parameter updates, data migrations, authz authorisation, etc,
// a new version-specific upgrade handler must be created.
func defaultUpgradeHandler(
	mm *module.Manager,
	_ *keepers.Keepers,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := cosmostypes.UnwrapSDKContext(ctx).Logger()
		logger.Info("Starting the migration in defaultUpgradeHandler")
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
