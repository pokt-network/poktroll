package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/pocket"
)

const (
	Upgrade_0_1_7_PlanName = "v0.1.7"

	newTokenSupplyAmount = int64(100_000_000_000_000_000) // 100B
)

// Upgrade_0_1_7 handles the upgrade to release `v0.1.7`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory footprint when iterating over Suppliers and Applications.
// https://github.com/pokt-network/poktroll/compare/v0.1.6..99c393
var Upgrade_0_1_7 = Upgrade{
	PlanName: Upgrade_0_1_7_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		newTokenSupplyCoins := cosmostypes.NewCoins(cosmostypes.NewInt64Coin(pocket.DenomMACT, newTokenSupplyAmount))

		mintNewTokenTypeMACT := func(ctx context.Context) (err error) {
			return keepers.BankKeeper.MintCoins(ctx, banktypes.ModuleName, newTokenSupplyCoins)
		}

		distributeNewTokenTypeMACT := func(ctx context.Context) (err error) {
			granteeAddr := NetworkAuthzGranteeAddress[cosmostypes.UnwrapSDKContext(ctx).ChainID()]
			granteeAccAddr := cosmostypes.MustAccAddressFromBech32(granteeAddr)

			return keepers.BankKeeper.SendCoinsFromModuleToAccount(ctx, banktypes.ModuleName, granteeAccAddr, newTokenSupplyCoins)
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := mintNewTokenTypeMACT(ctx); err != nil {
				return vm, err
			}

			if err := distributeNewTokenTypeMACT(ctx); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
