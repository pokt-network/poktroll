package upgrades

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
)

// Upgrade_0_0_10 is the upgrade handler for v0.0.10 Alpha TestNet upgrade
// Before/after validations should be done using the correct version (e.g. before - v0.0.9, after - v0.0.10)
var Upgrade_0_0_10 = Upgrade{
	PlanName: "v0.0.10",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			//
			// Add missing parameters and changes from `config.yml`
			// https://github.com/pokt-network/poktroll/compare/v0.0.9-3...96a9d29#diff-5a7db8dbadaef1b1b5a8738ba70b5ffac82b8e243732154165911284e08aad4b
			//

			// Add application min stake
			// Validate with: `poktrolld q application params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			appParams := keepers.ApplicationKeeper.GetParams(ctx)
			newMinStakeApp := cosmosTypes.NewCoin("upokt", math.NewInt(100000000))
			appParams.MinStake = &newMinStakeApp
			err := keepers.ApplicationKeeper.SetParams(ctx, appParams)
			if err != nil {
				return vm, err
			}

			// Add supplier min stake
			// Validate with: `poktrolld q supplier params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			supplierParams := keepers.SupplierKeeper.GetParams(ctx)
			newMinStakeSupplier := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			supplierParams.MinStake = &newMinStakeSupplier
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				return vm, err
			}

			// Add gateway min stake
			// Validate with: `poktrolld q gateway params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			gatewayParams := keepers.GatewayKeeper.GetParams(ctx)
			newMinStakeGW := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			gatewayParams.MinStake = &newMinStakeGW
			err = keepers.GatewayKeeper.SetParams(ctx, gatewayParams)
			if err != nil {
				return vm, err
			}

			// Adjust proof module parameters
			// Validate with: `poktrolld q proof params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			proofParams := keepers.ProofKeeper.GetParams(ctx)
			newProofRequirementThreshold := cosmosTypes.NewCoin("upokt", math.NewInt(20000000))
			newProofMissingPenalty := cosmosTypes.NewCoin("upokt", math.NewInt(320000000))
			proofParams.ProofRequirementThreshold = &newProofRequirementThreshold
			proofParams.ProofMissingPenalty = &newProofMissingPenalty
			err = keepers.ProofKeeper.SetParams(ctx, proofParams)
			if err != nil {
				return vm, err
			}

			// Add new shared module params
			// Validate with: `poktrolld q shared params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sharedParams.SupplierUnbondingPeriodSessions = uint64(1)
			sharedParams.ApplicationUnbondingPeriodSessions = uint64(1)
			sharedParams.ComputeUnitsToTokensMultiplier = uint64(42)
			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				return vm, err
			}

			//
			// Add new authz authorizations:
			// https://github.com/pokt-network/poktroll/compare/v0.0.9-3...96a9d29#diff-1698f4aae5353dd42a159c9dad0eca886805d5dc792a55a14f785f3b5ea767ee
			//

			// Validate before after with:
			// `poktrolld q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=https://testnet-validated-validator-rpc.poktroll.com/`
			newAuthorizations := []grantAuthorization{
				newTestNetGrantAuthorization("/poktroll.gateway.MsgUpdateParam"),
				newTestNetGrantAuthorization("/poktroll.application.MsgUpdateParam"),
				newTestNetGrantAuthorization("/poktroll.supplier.MsgUpdateParam"),
			}
			for _, authorization := range newAuthorizations {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					authorization.grantee,
					authorization.granter,
					authorization.authorization,
					authorization.expiration,
				)
				if err != nil {
					return vm, err
				}
			}

			// Seems like RelayMiningDifficulty have been moved from `tokenomics` to `services`. Do we need to move
			// and delete old RelayMiningDifficulty data? TODO if so.

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
