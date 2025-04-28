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

const (
	Upgrade_0_1_9_PlanName = "v0.1.9"
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

// Upgrade_0_1_9 handles the upgrade to release `v0.1.9`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory footprint when iterating over Suppliers and Applications.
// https://github.com/pokt-network/poktroll/compare/v0.1.6..99c393
var Upgrade_0_1_9 = Upgrade{
	PlanName: Upgrade_0_1_9_PlanName,
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
