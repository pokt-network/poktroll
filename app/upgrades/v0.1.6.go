package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/volatile"
)

const (
	Upgrade_0_1_6_PlanName = "v0.1.6"

	newTokenSupplyAmount = int64(100000000000) // 100B
)

// Upgrade_0_1_6 handles the upgrade to release `v0.1.6`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory footprint when iterating over Suppliers and Applications.
// TODO_IN_THIS_COMMIT: append commit hash to the range in the following comment, once known.
// https://github.com/pokt-network/poktroll/compare/v0.1.5..
var Upgrade_0_1_6 = Upgrade{
	PlanName: Upgrade_0_1_6_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		newTokenSupplyCoins := cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomMACT, newTokenSupplyAmount))

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
