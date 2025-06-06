// vNEXT_Template.go - Canonical Upgrade Template
//
// ────────────────────────────────────────────────────────────────
// TEMPLATE PURPOSE:
//   - This file is the canonical TEMPLATE for all future onchain upgrade files in the poktroll repo.
//   - DO NOT add upgrade-specific logic or changes to this file.
//   - YOU SHOULD NEVER NEED TO CHANGE THIS FILE
//
// USAGE INSTRUCTIONS:
//  1. To start a new upgrade cycle, rename vNEXT.go to the target version (e.g., v0.1.14.go) and update all identifiers accordingly:
//     cp ./app/upgrades/vNEXT.go ./app/upgrades/v0.1.14.go
//  2. Then, copy this file to vNEXT.go:
//     cp ./app/upgrades/vNEXT_Template.go ./app/upgrades/vNEXT.go
//  3. Look for the word "Template" in `vNEXT.go` and replace it with an empty string.
//  4. Make all upgrade-specific changes in vNEXT.go only.
//  5. To reset, restore, or start a new upgrade cycle, repeat fromstep 1.
//  6. Update the last entry in the `allUpgrades` slice in `app/upgrades.go` to point to the new upgrade version variable.
//
// vNEXT_Template.go should NEVER be modified for upgrade-specific logic.
// Only update this file to improve the template itself.
//
//	See also: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT
//
// ────────────────────────────────────────────────────────────────
package upgrades

import (
	"context"
	"time"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

var (
	// TODO_IN_THIS_COMMIT: this is a funciton of block time, which is per network!
	IbcConnectionParamMasExpectedTimePerBlock   = uint64((15 * time.Minute).Nanoseconds())
	IbcChannelParamUpgradeTimeoutRevisionNumber = uint64(0)
	IbcChannelParamUpgradeTimeoutRevisionHeight = uint64(0)
	IbcChannelParamUpgradeTimeoutTimestamp      = uint64(0)
	IbcClientParamAllowedClients                = []string{"07-tendermint"}

	IbcTransferParamSendEnabled    = true
	IbcTransferParamReceiveEnabled = true

	// Enable both ICA host and controller.
	IbcIcaHostParamHostEnabled             = true
	IbcIcaControllerParamControllerEnabled = true

	// Allow all messages to be executed via interchain accounts.
	IbcIcaHostParamAllowMessages = []string{"*"}
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		ibcConnectionParams := connectiontypes.Params{
			MaxExpectedTimePerBlock: IbcConnectionParamMasExpectedTimePerBlock,
		}

		ibcChannelParams := channeltypes.Params{
			UpgradeTimeout: channeltypes.Timeout{
				Height: ibcclienttypes.Height{
					RevisionNumber: IbcChannelParamUpgradeTimeoutRevisionNumber,
					RevisionHeight: IbcChannelParamUpgradeTimeoutRevisionHeight,
				},
				Timestamp: IbcChannelParamUpgradeTimeoutTimestamp,
			},
		}

		ibcClientParams := ibcclienttypes.Params{
			AllowedClients: IbcClientParamAllowedClients,
		}

		ibcTransferParams := ibctransfertypes.Params{
			SendEnabled:    IbcTransferParamSendEnabled,
			ReceiveEnabled: IbcTransferParamReceiveEnabled,
		}

		ibcIcaHostParams := icahosttypes.Params{
			HostEnabled:   IbcIcaHostParamHostEnabled,
			AllowMessages: IbcIcaHostParamAllowMessages,
		}

		ibcIcaControllerParams := icacontrollertypes.Params{
			ControllerEnabled: IbcIcaControllerParamControllerEnabled,
		}

		populateIBCParams := func(ctx context.Context) (err error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

			// IBC core
			keepers.IBCKeeper.ConnectionKeeper.SetParams(sdkCtx, ibcConnectionParams)
			keepers.IBCKeeper.ChannelKeeper.SetParams(sdkCtx, ibcChannelParams)
			keepers.IBCKeeper.ClientKeeper.SetParams(sdkCtx, ibcClientParams)

			// IBC transfer
			keepers.TransferKeeper.SetParams(sdkCtx, ibcTransferParams)

			// IBC interchain accounts host & controller
			keepers.ICAHostKeeper.SetParams(sdkCtx, ibcIcaHostParams)
			keepers.ICAControllerKeeper.SetParams(sdkCtx, ibcIcaControllerParams)

			return nil
		}

		bindIcaHostPort := func(ctx context.Context) (err error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			if !keepers.IBCKeeper.PortKeeper.IsBound(sdkCtx, icahosttypes.SubModuleName) {
				_ = keepers.IBCKeeper.PortKeeper.BindPort(sdkCtx, icahosttypes.SubModuleName)
			}
			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := populateIBCParams(ctx); err != nil {
				return vm, err
			}

			if err := bindIcaHostPort(ctx); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
