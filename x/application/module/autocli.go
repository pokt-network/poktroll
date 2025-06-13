package application

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	applicationtypes "github.com/pokt-network/poktroll/x/application/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           applicationtypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 					Long: `Shows all the parameters related to the application module.
				//
				// Example:
				// $ pocketd q application params`,
				// 				},
				// 				{
				// 					RpcMethod: "AllApplications",
				// 					Use:       "list-application",
				// 					Short:     "List all application",
				// 					Long: `List all the applications that staked in the network.
				//
				// Example:
				// $ pocketd q application list-application`,
				// 				},
				// 				{
				// 					RpcMethod: "Application",
				// 					Use:       "show-application [address]",
				// 					Short:     "Shows a application",
				// 					Long: `Finds a staked application given its address.
				//
				// Example:
				// $ pocketd q application show-application $(APP_ADDRESS)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				// 				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              applicationtypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// TODO_IN_THIS_COMMIT: update comment about skipping beucause authority gated...
				// TODO_IN_THIS_COMMIT: update comment... explain that commenting is the new skipping,
				// and skipping is how we use AutoCLI with TX commands because we have to preempt it in order to register
				// custom flags. This means that we're creating the command, not autoCLI; therefore,
				// we need to skip it. We still use these conventional autoCLI data structures to
				// express the integration conventionally (save for the skips).
				// TODO_IN_THIS_COMMIT: consolidate existing custom commands with the commented ones.
				// Custom commands SHOULD be "justified"; i.e., AutoCLI integration is insufficient
				// for some reason. For example, a command is authority gated or requires non-trivial
				// custom logic like signature verification.
				//              {
				//              	RpcMethod: "UpdateParams",
				//              	GovProposal: true,
				//              	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				//              	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				//              },
				// 				{
				// 					RpcMethod: "StakeApplication",
				// 					Use:       "stake-application [stake] [services]",
				// 					Short:     "Send a stake-application tx",
				// 					Long: `Stake an application using a config file. This is a broadcast operation that will stake
				// the tokens and serviceIds and associate them with the application specified by the 'from' address.
				//
				// Example:
				// $ pocketd tx application stake-application --config app_stake_config.yaml --keyring-backend test --from $(APP)`,
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
				// $ pocketd tx application unstake-application --keyring-backend test --from $(APP)`,
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
				// $ pocketd tx application delegate-to-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP)`,
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
				// $ pocketd tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP)`,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gateway_address"}},
				// 				},
				{
					RpcMethod:      "TransferApplication",
					Use:            "transfer [source app address] [destination app address]",
					Short:          "Transfer the application from [source app address] to [destination app address] and remove the source application",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "source_address"}, {ProtoField: "destination_address"}},
					// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
					Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				},
				// {
				// 	RpcMethod:      "UpdateParam",
				// 	Use:            "update-param [name] [as-type]",
				// 	Short:          "Send a update-param tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				// 	GovProposal:    true,
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
