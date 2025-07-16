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
	Upgrade_NEXT_PlanName = "vNEXT"
)

const (
	// BlockTimeAdjustmentFactor adds a safety margin to estimated block time
	// for calculating IBC parameters:
	// - ibc.connection.params.max_expected_time_per_block
	// - ibc.channel.params.upgrade_timeout.timestamp
	BlockTimeAdjustmentFactor = 1.5

	// IBC channel upgrade timeout revision settings
	IbcChannelParamUpgradeTimeoutRevisionNumber = uint64(0)
	IbcChannelParamUpgradeTimeoutRevisionHeight = uint64(0)

	// IBC transfer settings
	IbcTransferParamSendEnabled    = true
	IbcTransferParamReceiveEnabled = true

	// ICA (Interchain Accounts) settings
	// Ref: https://ibc.cosmos.network/v8/apps/interchain-accounts/parameters/
	IbcIcaHostParamHostEnabled             = true
	IbcIcaControllerParamControllerEnabled = true

	// Format string for parsing durations in seconds
	durationSecondsFormat = "%vs"
)

var (
	// Allowed IBC client types
	// Ref: https://ibc.cosmos.network/params/params.md/
	IbcClientParamAllowedClients = []string{"07-tendermint"}

	// Allow all message types for ICA execution
	IbcIcaHostParamAllowMessages = []string{"*"}
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// Changes:
// - Updates to the Morse account recovery allowlist
// - Sets all IBC parameters to enable IBC support
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No store migrations in this upgrade
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		setIBCParams := func(ctx context.Context) (err error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger().With("upgrade_plan_name", Upgrade_NEXT_PlanName)
			logger.Info("Starting IBC parameter configuration")

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
			logger.Info("Setting IBC connection params", "params", ibcConnectionParams)

			// IBC channel params
			upgradeTimeoutSeconds := estimatedBlockDuration.Seconds() * 4 * BlockTimeAdjustmentFactor

			// Channel upgrade timeout calculation:
			// - Handshake steps: INIT → TRY → ACK → CONFIRM (4 blocks minimum)
			// - Formula: timeout ≥ 4 × slower_chain_block_time × adjustment_factor
			// - Localnet recommendation: 60-300 seconds
			// - Zero value invalid unless height-based timeout is set
			ibcChannelParamUpgradeTimeoutTimestamp, err := time.ParseDuration(
				fmt.Sprintf(durationSecondsFormat, upgradeTimeoutSeconds),
			)
			if err != nil {
				return err
			}

			// Ref: https://ibc.cosmos.network/params/params.md/
			ibcChannelParams := channeltypes.Params{
				UpgradeTimeout: channeltypes.Timeout{
					Height: ibcclienttypes.Height{
						RevisionNumber: IbcChannelParamUpgradeTimeoutRevisionNumber,
						RevisionHeight: IbcChannelParamUpgradeTimeoutRevisionHeight,
					},
					Timestamp: uint64(ibcChannelParamUpgradeTimeoutTimestamp.Nanoseconds()),
				},
			}
			logger.Info("Setting IBC channel params", "params", ibcChannelParams)

			// IBC client params
			ibcClientParams := ibcclienttypes.Params{
				AllowedClients: IbcClientParamAllowedClients,
			}
			logger.Info("Setting IBC client params", "params", ibcClientParams)

			// IBC transfer params
			// Ref: https://ibc.cosmos.network/v8/apps/transfer/params/
			ibcTransferParams := ibctransfertypes.Params{
				SendEnabled:    IbcTransferParamSendEnabled,
				ReceiveEnabled: IbcTransferParamReceiveEnabled,
			}
			logger.Info("Setting IBC transfer params", "params", ibcTransferParams)

			// IBC interchain accounts host params
			ibcIcaHostParams := icahosttypes.Params{
				HostEnabled:   IbcIcaHostParamHostEnabled,
				AllowMessages: IbcIcaHostParamAllowMessages,
			}
			logger.Info("Setting IBC interchain accounts host params", "params", ibcIcaHostParams)

			// IBC interchain accounts controller params
			ibcIcaControllerParams := icacontrollertypes.Params{
				ControllerEnabled: IbcIcaControllerParamControllerEnabled,
			}
			logger.Info("Setting IBC interchain accounts controller params", "params", ibcIcaControllerParams)

			// Set IBC core params (connection, channel, client)
			keepers.IBCKeeper.ConnectionKeeper.SetParams(sdkCtx, ibcConnectionParams)
			keepers.IBCKeeper.ChannelKeeper.SetParams(sdkCtx, ibcChannelParams)
			keepers.IBCKeeper.ClientKeeper.SetParams(sdkCtx, ibcClientParams)

			// Set IBC transfer params
			keepers.TransferKeeper.SetParams(sdkCtx, ibcTransferParams)

			// Set IBC interchain accounts host & controller params
			keepers.ICAHostKeeper.SetParams(sdkCtx, ibcIcaHostParams)
			keepers.ICAControllerKeeper.SetParams(sdkCtx, ibcIcaControllerParams)

			// Initialize IBC client sequence counter
			keepers.IBCKeeper.ClientKeeper.SetNextClientSequence(sdkCtx, 0)

			// Initialize IBC connection sequence counter to fix "next connection sequence is nil" error
			keepers.IBCKeeper.ConnectionKeeper.SetNextConnectionSequence(sdkCtx, 0)

			// Initialize IBC channel sequence counter to fix potential "next channel sequence is nil" error
			keepers.IBCKeeper.ChannelKeeper.SetNextChannelSequence(sdkCtx, 0)

			// Initialize the transfer module for IBC support
			// During genesis, the transfer module's InitGenesis sets up port binding
			// We need to replicate this during upgrade
			transferGenesis := ibctransfertypes.GenesisState{
				PortId:      ibctransfertypes.PortID,
				DenomTraces: []ibctransfertypes.DenomTrace{},
				Params: ibctransfertypes.Params{
					SendEnabled:    IbcTransferParamSendEnabled,
					ReceiveEnabled: IbcTransferParamReceiveEnabled,
				},
				TotalEscrowed: cosmostypes.NewCoins(),
			}

			// Call InitGenesis to properly initialize the transfer module
			keepers.TransferKeeper.InitGenesis(sdkCtx, transferGenesis)
			logger.Info("Successfully completed IBC parameter configuration")

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_NEXT_PlanName)
			logger.Info("Starting upgrade handler")

			if err := setIBCParams(ctx); err != nil {
				logger.Error("Failed to set IBC parameters", "error", err)
				return vm, err
			}

			logger.Info("Successfully completed upgrade handler")
			return vm, nil
		}
	},
}
