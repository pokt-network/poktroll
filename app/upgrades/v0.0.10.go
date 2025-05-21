package upgrades

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// Upgrade_0_0_10 is the upgrade handler for v0.0.10 Alpha TestNet upgrade
// Before/after validations should be done using the correct version, mimiching real-world scenario:
//   - Before: v0.0.9
//   - After: v0.0.10
var Upgrade_0_0_10 = Upgrade{
	PlanName: "v0.0.10",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		// Adds new parameters using ignite's config.yml as a reference. Assuming we don't need any other parameters.
		// https://github.com/pokt-network/poktroll/compare/v0.0.9-3...ff76430
		applyNewParameters := func(ctx context.Context) (err error) {
			// Add application min stake
			// Validate with: `pocketd q application params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			appParams := keepers.ApplicationKeeper.GetParams(ctx)
			newMinStakeApp := cosmostypes.NewCoin("upokt", math.NewInt(100000000))
			appParams.MinStake = &newMinStakeApp
			err = keepers.ApplicationKeeper.SetParams(ctx, appParams)
			if err != nil {
				return err
			}

			// Add supplier min stake
			// Validate with: `pocketd q supplier params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			supplierParams := keepers.SupplierKeeper.GetParams(ctx)
			newMinStakeSupplier := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
			supplierParams.MinStake = &newMinStakeSupplier
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				return err
			}

			// Add gateway min stake
			// Validate with: `pocketd q gateway params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			gatewayParams := keepers.GatewayKeeper.GetParams(ctx)
			newMinStakeGW := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
			gatewayParams.MinStake = &newMinStakeGW
			err = keepers.GatewayKeeper.SetParams(ctx, gatewayParams)
			if err != nil {
				return err
			}

			// Adjust proof module parameters
			// Validate with: `pocketd q proof params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			newProofRequirementThreshold := cosmostypes.NewCoin("upokt", math.NewInt(20000000))
			newProofMissingPenalty := cosmostypes.NewCoin("upokt", math.NewInt(320000000))
			newProofSubmissionFee := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
			proofParams := prooftypes.NewParams(
				0.25,
				&newProofRequirementThreshold,
				&newProofMissingPenalty,
				&newProofSubmissionFee,
			)

			err = keepers.ProofKeeper.SetParams(ctx, proofParams)
			if err != nil {
				return err
			}

			// Add new shared module params
			// Validate with: `pocketd q shared params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sharedParams.SupplierUnbondingPeriodSessions = uint64(1)
			sharedParams.ApplicationUnbondingPeriodSessions = uint64(1)
			sharedParams.ComputeUnitsToTokensMultiplier = uint64(42)
			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				return err
			}
			return
		}

		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.0.9-3...ff76430
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// `pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=https://testnet-validated-validator-rpc.poktroll.com/`
			grantAuthorizationMessages := []string{
				"/pocket.gateway.MsgUpdateParam",
				"/pocket.application.MsgUpdateParam",
				"/pocket.supplier.MsgUpdateParam",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmostypes.AccAddress(AlphaTestNetPnfAddress),
					cosmostypes.AccAddress(AlphaTestNetAuthorityAddress),
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
			}
			return
		}

		// Returns the upgrade handler for v0.0.10
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			err := applyNewParameters(ctx)
			if err != nil {
				return vm, err
			}

			err = applyNewAuthorizations(ctx)
			if err != nil {
				return vm, err
			}

			// RelayMiningDifficulty have been moved from `tokenomics` to `services` in this diff.
			// Ideally (in prod), we should have migrated the data before removing query/msg types in `tokenomics` module.
			// In practice (development), we don't want to spend time on it.
			// Since we know that new RelayMiningDifficulty will be re-created and real tokens are not in danger,
			// we can skip this step now.

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
