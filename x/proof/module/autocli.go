package proof

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           prooftypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				//				{
				//					RpcMethod: "Params",
				//					Use:       "params",
				//					Short:     "Shows the parameters of the module",
				//				},
				//				{
				//					RpcMethod: "AllClaims",
				//					Use:       "list-claim",
				//					Short:     "List all claim",
				//				},
				//				{
				//					RpcMethod:      "Claim",
				//					Use:            "show-claim [id]",
				//					Short:          "Shows a claim",
				//					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				//				},
				//				{
				//					RpcMethod: "AllProofs",
				//					Use:       "list-proof",
				//					Short:     "List all proof",
				//				},
				//				{
				//					RpcMethod:      "Proof",
				//					Use:            "show-proof [id]",
				//					Short:          "Shows a proof",
				//					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				//				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              prooftypes.Msg_serviceDesc.ServiceName,
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
				// 	RpcMethod:      "CreateClaim",
				// 	Use:            "create-claim [session-header] [root-hash]",
				// 	Short:          "Send a create-claim tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "sessionHeader"}, {ProtoField: "rootHash"}},
				// },
				// {
				// 	RpcMethod:      "SubmitProof",
				// 	Use:            "submit-proof [session-header] [proof]",
				// 	Short:          "Send a submit-proof tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "sessionHeader"}, {ProtoField: "proof"}},
				// },
				// {
				// 	RpcMethod: "UpdateParam",
				// 	Use:       "update-param [name] [value]",
				// 	Short:     "Send a update-param tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "as_type"}},
				// 	GovProposal: true,
				// 	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				// 	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
