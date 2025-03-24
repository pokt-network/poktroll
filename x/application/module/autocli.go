package application

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/pocket/application"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 					Long: `Shows all the parameters related to the application module.
				//
				// Example:
				// $ pocketd q application params --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 				},
				// 				{
				// 					RpcMethod: "AllApplications",
				// 					Use:       "list-application",
				// 					Short:     "List all application",
				// 					Long: `List all the applications that staked in the network.
				//
				// Example:
				// $ pocketd q application list-application --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 				},
				// 				{
				// 					RpcMethod: "Application",
				// 					Use:       "show-application [address]",
				// 					Short:     "Shows a application",
				// 					Long: `Finds a staked application given its address.
				//
				// Example:
				// $ pocketd q application show-application $(APP_ADDRESS) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				// 				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "UpdateParams",
				// 					Skip:      true, // skipped because authority gated
				// 				},
				// 				{
				// 					RpcMethod: "StakeApplication",
				// 					Use:       "stake-application [stake] [services]",
				// 					Short:     "Send a stake-application tx",
				// 					Long: `Stake an application using a config file. This is a broadcast operation that will stake
				// the tokens and serviceIds and associate them with the application specified by the 'from' address.
				//
				// Example:
				// $ pocketd tx application stake-application --config app_stake_config.yaml --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				// 				},
				// 				{
				// 					RpcMethod: "UnstakeApplication",
				// 					Use:       "unstake-application",
				// 					Short:     "Send a unstake-application tx",
				// 					Long: `Unstake an application. This is a broadcast operation that will unstake
				// the application specified by the 'from' address.
				//
				// Example:
				// $ pocketd tx application unstake-application --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				// 				},
				// 				{
				// 					RpcMethod: "DelegateToGateway",
				// 					Use:       "delegate-to-gateway [gateway-address]",
				// 					Short:     "Send a delegate-to-gateway tx",
				// 					Long: `Delegate an application to the gateway with the provided address. This is a broadcast operation
				// that delegates authority to the gateway specified to sign relays requests for the application, allowing the gateway
				// act on the behalf of the application during a session.
				//
				// Example:
				// $ pocketd tx application delegate-to-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gateway_address"}},
				// 				},
				// 				{
				// 					RpcMethod: "UndelegateFromGateway",
				// 					Use:       "undelegate-from-gateway [gateway-address]",
				// 					Short:     "Send a undelegate-from-gateway tx",
				// 					Long: `Undelegate an application from the gateway with the provided address. This is a broadcast operation
				// that removes the authority from the gateway specified to sign relays requests for the application, disallowing the gateway
				// act on the behalf of the application during a session.
				//
				// Example:
				// $ pocketd tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gateway_address"}},
				// 				},
				{
					RpcMethod:      "TransferApplication",
					Use:            "transfer [source app address] [destination app address]",
					Short:          "Transfer the application from [source app address] to [destination app address] and remove the source application",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "source_address"}, {ProtoField: "destination_address"}},
				},
				//{
				//	RpcMethod:      "UpdateParam",
				//	Use:            "update-param [name] [as-type]",
				//	Short:          "Send a update-param tx",
				//	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				//},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
