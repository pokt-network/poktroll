package tokenomics

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           tokenomicstypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 					Long: `Shows all the parameters related to the tokenomics module.
				//
				// Example:
				// $ pocketd q tokenomics params
				// 				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              tokenomicstypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				// TODO_IN_THIS_COMMIT: update comment about skipping beucause authority gated...
				// TODO_IN_THIS_COMMIT: update comment... explain that commenting is the new skipping,
				// and skipping is how we use AutoCLI with TX commands because we have to preempt it in order to register
				// custom flags. This means that we're creating the command, not autoCLI; therefore,
				// we need to skip it. We still use these conventional autoCLI data structures to
				// express the integration conventionally (save for the skips).
				// {
				// 	RpcMethod:   "UpdateParams",
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
