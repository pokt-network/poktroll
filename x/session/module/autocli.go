package session

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           sessiontypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 				},
				// 				{
				// 					RpcMethod: "GetSession",
				// 					Use:       "get-session [application-address] [service] [block-height]",
				// 					Short:     "Query get-session",
				// 					Long: `Query the session data for a specific (app, service, height) tuple.
				//
				// This is a query operation that will not result in a state transition but simply gives a view into the chain state.
				//
				// Example:
				// $ pocketd q session get-session pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 svc1 42
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "application_address"}, {ProtoField: "service"}, {ProtoField: "block_height"}},
				// 				},

				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              sessiontypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
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
				// {
				// 	RpcMethod: "UpdateParams",
				// 	GovProposal: true,
				// 	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				// 	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				// },
				// {
				// 	RpcMethod:      "UpdateParam",
				// 	Use:            "update-param [name] [as-type]",
				// 	Short:          "Send a update-param tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				// 	GovProposal: true,
				// 	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				// 	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
