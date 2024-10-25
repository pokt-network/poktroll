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
			appParams := keepers.ApplicationKeeper.GetParams(ctx)
			newMinStakeApp := cosmosTypes.NewCoin("upokt", math.NewInt(100000000))
			appParams.MinStake = &newMinStakeApp
			err := keepers.ApplicationKeeper.SetParams(ctx, appParams)
			if err != nil {
				return vm, err
			}

			// Add supplier min stake
			supplierParams := keepers.SupplierKeeper.GetParams(ctx)
			newMinStakeSupplier := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			supplierParams.MinStake = &newMinStakeSupplier
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				return vm, err
			}

			// Add gateway min stake
			gatewayParams := keepers.GatewayKeeper.GetParams(ctx)
			newMinStakeGW := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			gatewayParams.MinStake = &newMinStakeGW
			err = keepers.GatewayKeeper.SetParams(ctx, gatewayParams)
			if err != nil {
				return vm, err
			}

			// Adjust proof module parameters
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
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sharedParams.SupplierUnbondingPeriodSessions = uint64(1)
			sharedParams.ApplicationUnbondingPeriodSessions = uint64(1)
			sharedParams.ComputeUnitsToTokensMultiplier = uint64(42)
			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				return vm, err
			}

			// // Get current consensus module parameters
			// currentParams, err := keepers.ConsensusParamsKeeper.ParamsStore.Get(ctx)
			// if err != nil {
			// 	return vm, err
			// }

			// // Supply all params even when changing just one, as `ToProtoConsensusParams` requires them to be present.
			// newParams := consensusparamtypes.MsgUpdateParams{
			// 	Authority: keepers.ConsensusParamsKeeper.GetAuthority(),
			// 	Block:     currentParams.Block,
			// 	Evidence:  currentParams.Evidence,
			// 	Validator: currentParams.Validator,

			// 	// This seems to be deprecated/not needed, but it's fine as we're copying the existing data.
			// 	Abci: currentParams.Abci,
			// }

			// // Increase block size two-fold, 22020096 is the default value.
			// newParams.Block.MaxBytes = 22020096 * 2

			// // Update the chain state
			// if _, err = keepers.ConsensusParamsKeeper.UpdateParams(ctx, &newParams); err != nil {
			// 	return vm, err
			// }

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
