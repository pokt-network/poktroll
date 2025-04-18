package upgrades

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
)

const Upgrade_0_1_4_PlanName = "v0.1.4-test1"

// Upgrade_0_1_4 handles the upgrade to release `v0.1.4`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets as well as Mainnet.
var Upgrade_0_1_4 = Upgrade{
	// Plan Name
	PlanName: Upgrade_0_1_4_PlanName,

	// No migrations or state changes in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.1.3..UPDATE_ME
		applyNewAuthorizations := func(ctx context.Context, upgradeLogger log.Logger) (err error) {
			logger := upgradeLogger.With("method", "applyNewAuthorizations")
			logger.Info("Starting authorization updates")

			// Validate before/after with:
			// 	pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=<network_rpc_url>
			// Find a network RPC URL here: https://dev.poktroll.com/category/explorers-faucets-wallets-and-more
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParams",
				"/pocket.migration.MsgImportMorseClaimableAccounts",
			}

			// expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			expiration := time.Now().AddDate(100, 0, 0)
			// if err != nil {
			// 	return fmt.Errorf("failed to parse time: %w", err)
			// }

			// Get the granter address of the migration module (i.e. authority)
			granterAddr := "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t" //keepers.MigrationKeeper.GetAuthority()

			// Get the grantee address for the current network (i.e. pnf or grove)
			granteeAddr := "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" //NetworkAuthzGranteeAddress[cosmosTypes.UnwrapSDKContext(ctx).ChainID()]

			logger.Info(fmt.Sprintf("OLSH2222  AUTHZ KEEPER: %p", &keepers.AuthzKeeper))

			authorizations, err := keepers.AuthzKeeper.GetAuthorizations(ctx, cosmosTypes.AccAddress(granteeAddr), cosmosTypes.AccAddress(granterAddr))
			if err != nil {
				return fmt.Errorf("failed to get authorizations: %w", err)
			}
			logger.Info(fmt.Sprintf("CURRENT number of authorizations: %d", len(authorizations)))

			// if entry.Expiration != nil && entry.Expiration.Before(now) {
			// 	continue
			// }

			granteeCosmosAddr, err := keepers.AccountKeeper.AddressCodec().StringToBytes(granteeAddr)
			if err != nil {
				panic(err)
			}
			granterCosmosAddr, err := keepers.AccountKeeper.AddressCodec().StringToBytes(granterAddr)
			if err != nil {
				panic(err)
			}

			// a := authz.NewGenericAuthorization(msg)

			// err = keepers.AuthzKeeper.SaveGrant(ctx, cosmosTypes.AccAddress(granteeCosmosAddr), cosmosTypes.AccAddress(granter), a, &expiration)
			// if err != nil {
			// 	panic(err)
			// }

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					granteeCosmosAddr,
					granterCosmosAddr,
					// cosmosTypes.AccAddress(granteeCosmosAddr),
					// cosmosTypes.AccAddress(granterAddr),
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
				logger.Info(fmt.Sprintf("Generic authorization granted for message %s from %s to %s", msg, granterAddr, granteeAddr))
			}

			authorizations, err = keepers.AuthzKeeper.GetAuthorizations(ctx, cosmosTypes.AccAddress(granteeAddr), cosmosTypes.AccAddress(granterAddr))
			if err != nil {
				return fmt.Errorf("failed to get authorizations: %w", err)
			}
			logger.Info(fmt.Sprintf("NEW number of authorizations: %d", len(authorizations)))
			logger.Info("Successfully finished authorization updates")
			return
		}

		// Returns the upgrade handler for v0.1.4
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_4_PlanName)
			logger.Info("Starting upgrade handler")

			// Apply authorization changes
			logger.Info("Starting authorization updates")
			err := applyNewAuthorizations(ctx, logger)
			if err != nil {
				logger.Error("Failed to apply new authorizations", "error", err)
				return vm, err
			}
			logger.Info("Successfully completed authorization updates")

			logger.Info("Starting module migrations section")
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations", "error", err)
				return vm, err
			}
			logger.Info("Successfully completed module migrations")

			logger.Info("Successfully completed upgrade")
			return vm, nil
		}
	},
}
