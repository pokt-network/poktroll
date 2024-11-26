package upgrades

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/pokt-network/poktroll/app/keepers"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Upgrade_0_0_11 is the upgrade handler for v0.0.11 Alpha TestNet upgrade
// Beta TestNet was launched with v0.0.11, so this upgrade is exclusively for Alpha TestNet.
//   - Before: v0.0.10
//   - After: v0.0.11
var Upgrade_0_0_11 = Upgrade{
	PlanName: "v0.0.11",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		// Adds new parameters using ignite's config.yml as a reference. Assuming we don't need any other parameters.
		// https://github.com/pokt-network/poktroll/compare/v0.0.10...v0.0.11-rc
		applyNewParameters := func(ctx context.Context) (err error) {
			// Add application min stake
			// Validate with: `poktrolld q application params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			appParams := keepers.ApplicationKeeper.GetParams(ctx)
			newMinStakeApp := cosmosTypes.NewCoin("upokt", math.NewInt(100000000))
			appParams.MinStake = &newMinStakeApp
			err = keepers.ApplicationKeeper.SetParams(ctx, appParams)
			if err != nil {
				return err
			}

			// Add supplier min stake
			// Validate with: `poktrolld q supplier params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			supplierParams := keepers.SupplierKeeper.GetParams(ctx)
			newMinStakeSupplier := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			supplierParams.MinStake = &newMinStakeSupplier
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				return err
			}

			// Add gateway min stake
			// Validate with: `poktrolld q gateway params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			gatewayParams := keepers.GatewayKeeper.GetParams(ctx)
			newMinStakeGW := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
			gatewayParams.MinStake = &newMinStakeGW
			err = keepers.GatewayKeeper.SetParams(ctx, gatewayParams)
			if err != nil {
				return err
			}

			// Adjust proof module parameters
			// Validate with: `poktrolld q proof params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			newProofRequirementThreshold := cosmosTypes.NewCoin("upokt", math.NewInt(20000000))
			newProofMissingPenalty := cosmosTypes.NewCoin("upokt", math.NewInt(320000000))
			newProofSubmissionFee := cosmosTypes.NewCoin("upokt", math.NewInt(1000000))
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
			// Validate with: `poktrolld q shared params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sharedParams.SupplierUnbondingPeriodSessions = uint64(1)
			sharedParams.ApplicationUnbondingPeriodSessions = uint64(1)
			sharedParams.ComputeUnitsToTokensMultiplier = uint64(42)
			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				return err
			}

			// Set num_suppliers_per_session to 15
			// Validate with: `poktrolld q session params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			sessionParams := keepers.SessionKeeper.GetParams(ctx)
			sessionParams.NumSuppliersPerSession = uint64(15)
			err = keepers.SessionKeeper.SetParams(ctx, sessionParams)
			if err != nil {
				return err
			}

			// Set tokenomics params:
			//   mint_allocation_percentages:
			//     dao: 0.1
			// 	   proposer: 0.05
			// 	   supplier: 0.7
			// 	   source_owner: 0.15
			//     application: 0.0
			//   # The dao reward address SHOULD match that of the "pnf" below (i.e. `make poktrolld_addr ACC_NAME=pnf`).
			//   # This is the address that will receive the dao/foundation rewards during claim settlement (global mint TLM).
			//   # TODO_MAINNET(@olshansk): Consolidate the usage of DAO/PNF throughout the configs & codebase.
			//   dao_reward_address: "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)
			tokenomicsParams.MintAllocationPercentages = tokenomicstypes.MintAllocationPercentages{
				Dao:         0.1,
				Proposer:    0.05,
				Supplier:    0.7,
				SourceOwner: 0.15,
				Application: 0.0,
			}
			tokenomicsParams.DaoRewardAddress = AlphaTestNetPnfAddress
			err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams)
			if err != nil {
				return err
			}
			return
		}

		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.0.10...v0.0.11-rc
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// `poktrolld q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=https://testnet-validated-validator-rpc.poktroll.com/`
			grantAuthorizationMessages := []string{
				"/poktroll.session.MsgUpdateParam",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmosTypes.AccAddress(AlphaTestNetPnfAddress),
					cosmosTypes.AccAddress(AlphaTestNetAuthorityAddress),
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
			}
			return
		}

		// Returns the upgrade handler for v0.0.11
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			err := applyNewParameters(ctx)
			if err != nil {
				return vm, err
			}

			err = applyNewAuthorizations(ctx)
			if err != nil {
				return vm, err
			}

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
