package upgrades

import (
	"context"
	"fmt"
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
	"github.com/pokt-network/poktroll/app/pocket"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "v99"
)

const (
	// BlockTimeAdjustmentFactor is a coeffient used to add a safety margin to the estimated block
	//time for use in calculating IBC parameters which are a function of the estimated block time:
	// - ibc.connection.params.max_expected_time_per_block
	// - ibc.channel.params.upgrade_timeout.timestamp
	BlockTimeAdjustmentFactor = 1.5

	// Only set during an IBC upgrade.
	IbcChannelParamUpgradeTimeoutRevisionNumber = uint64(0)
	IbcChannelParamUpgradeTimeoutRevisionHeight = uint64(0)

	// Enable IBC transfers (send & receive).
	IbcTransferParamSendEnabled    = true
	IbcTransferParamReceiveEnabled = true

	// Enable both ICA host and controller support.
	// See:
	// - https://ibc.cosmos.network/v8/apps/interchain-accounts/parameters/
	// - https://ibc.cosmos.network/v8/apps/interchain-accounts/integration/
	IbcIcaHostParamHostEnabled             = true
	IbcIcaControllerParamControllerEnabled = true

	// durationSecondsFormat is the format string used to parse durations in seconds.
	durationSecondsFormat = "%vs"
)

var (
	// See: https://ibc.cosmos.network/params/params.md/
	IbcClientParamAllowedClients = []string{"07-tendermint"}

	// Allow all messages to be executed via interchain accounts.
	IbcIcaHostParamAllowMessages = []string{"*"}
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// - Set all IBC related parameters to reasonable starting values (required for IBC support)
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
		setIBCParams := func(ctx context.Context) (err error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

			estimatedBlockDuration, isFound := pocket.EstimatedBlockDurationByChainId[sdkCtx.ChainID()]
			if !isFound {
				return fmt.Errorf("chain ID %s not found in EstimatedBlockDurationByChainId", sdkCtx.ChainID())
			}

			// IBC connection params
			maxExpectedBlockTimeSeconds := estimatedBlockDuration.Seconds() * BlockTimeAdjustmentFactor
			ibcConnectionParamMaxExpectedTimePerBlock, err := time.ParseDuration(
				fmt.Sprintf(durationSecondsFormat, maxExpectedBlockTimeSeconds),
			)
			if err != nil {
				return err
			}

			ibcConnectionParams := connectiontypes.Params{
				MaxExpectedTimePerBlock: uint64(ibcConnectionParamMaxExpectedTimePerBlock.Nanoseconds()),
			}
			fmt.Printf("IBC connection params: %+v\n", ibcConnectionParams)

			// IBC channel params
			upgradeTimeoutSeconds := estimatedBlockDuration.Seconds() * 4 * BlockTimeAdjustmentFactor

			// ibcChannelParamUpgradeTimeoutTimestamp defines the maximum allowed duration (in nanoseconds)
			// for completing an IBC channel upgrade handshake. The handshake involves multiple
			// steps (INIT → TRY → ACK → CONFIRM), each of which occurs in separate blocks on
			// each chain. As such, this timeout should account for at least 4 relayed IBC messages,
			// which means it must exceed:
			//
			//     timeout ≥ 4 × slower_chain_block_time × adjustment_factor
			//
			// For localnets, a value between 60–300 seconds (e.g. 300_000_000_000 ns) is typically
			// sufficient to accommodate relayer delays. A zero value is considered invalid unless
			// a non-zero height-based timeout is also set.
			ibcChannelParamUpgradeTimeoutTimestamp, err := time.ParseDuration(
				fmt.Sprintf(durationSecondsFormat, upgradeTimeoutSeconds),
			)
			if err != nil {
				return err
			}

			ibcChannelParams := channeltypes.Params{
				UpgradeTimeout: channeltypes.Timeout{
					Height: ibcclienttypes.Height{
						RevisionNumber: IbcChannelParamUpgradeTimeoutRevisionNumber,
						RevisionHeight: IbcChannelParamUpgradeTimeoutRevisionHeight,
					},
					Timestamp: uint64(ibcChannelParamUpgradeTimeoutTimestamp.Nanoseconds()),
				},
			}
			fmt.Printf("IBC channel params: %+v\n", ibcChannelParams)

			// IBC client params
			// See: https://ibc.cosmos.network/params/params.md/
			ibcClientParams := ibcclienttypes.Params{
				AllowedClients: IbcClientParamAllowedClients,
			}
			fmt.Printf("IBC client params: %+v\n", ibcClientParams)

			// IBC transfer params
			// See: https://ibc.cosmos.network/v8/apps/transfer/params/
			ibcTransferParams := ibctransfertypes.Params{
				SendEnabled:    IbcTransferParamSendEnabled,
				ReceiveEnabled: IbcTransferParamReceiveEnabled,
			}
			fmt.Printf("IBC transfer params: %+v\n", ibcTransferParams)

			// IBC interchain accounts host params
			ibcIcaHostParams := icahosttypes.Params{
				HostEnabled:   IbcIcaHostParamHostEnabled,
				AllowMessages: IbcIcaHostParamAllowMessages,
			}
			fmt.Printf("IBC interchain accounts host params: %+v\n", ibcIcaHostParams)

			// IBC interchain accounts controller params
			// See: https://ibc.cosmos.network/v8/apps/interchain-accounts/parameters/
			ibcIcaControllerParams := icacontrollertypes.Params{
				ControllerEnabled: IbcIcaControllerParamControllerEnabled,
			}
			fmt.Printf("IBC interchain accounts controller params: %+v\n", ibcIcaControllerParams)

			// Set IBC core params
			// - connection
			// - channel
			// - client
			keepers.IBCKeeper.ConnectionKeeper.SetParams(sdkCtx, ibcConnectionParams)
			keepers.IBCKeeper.ChannelKeeper.SetParams(sdkCtx, ibcChannelParams)
			keepers.IBCKeeper.ClientKeeper.SetParams(sdkCtx, ibcClientParams)

			// Initialize IBC client sequence to enable IBC client creation
			// This fixes the "next client sequence is nil" error when creating IBC clients
			keepers.IBCKeeper.ClientKeeper.SetNextClientSequence(sdkCtx, 0)

			// Initialize IBC connection sequence to enable IBC connection creation
			// This fixes the "next connection sequence is nil" error when creating IBC connections
			keepers.IBCKeeper.ConnectionKeeper.SetNextConnectionSequence(sdkCtx, 0)

			// Initialize IBC channel sequence to enable IBC channel creation
			// This fixes the "next channel sequence is nil" error when creating IBC channels
			keepers.IBCKeeper.ChannelKeeper.SetNextChannelSequence(sdkCtx, 0)

			// Set IBC transfer params
			keepers.TransferKeeper.SetParams(sdkCtx, ibcTransferParams)

			// Set IBC interchain accounts host & controller params
			keepers.ICAHostKeeper.SetParams(sdkCtx, ibcIcaHostParams)
			keepers.ICAControllerKeeper.SetParams(sdkCtx, ibcIcaControllerParams)

			// Bind transfer port to enable IBC token transfers
			// This fixes the "capability not found" error when creating transfer channels
			if !keepers.IBCKeeper.PortKeeper.IsBound(sdkCtx, ibctransfertypes.ModuleName) {
				_ = keepers.IBCKeeper.PortKeeper.BindPort(sdkCtx, ibctransfertypes.ModuleName)
			}

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := setIBCParams(ctx); err != nil {
				return vm, err
			}

			// Set IBC module version to prevent re-initialization via InitGenesis
			// This ensures IBC modules don't try to initialize again after upgrade
			vm["ibc"] = 1

			return vm, nil
		}
	},
}
