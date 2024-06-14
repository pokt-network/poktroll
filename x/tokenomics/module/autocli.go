package tokenomics

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/tokenomics"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 					Long: `Shows all the parameters related to the tokenomics module.
				//
				// Example:
				// $ poktrolld q tokenomics params --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
				// 				},
				{
					RpcMethod: "RelayMiningDifficultyAll",
					Use:       "list-relay-mining-difficulty",
					Short:     "List all relay-mining-difficulty",
				},
				{
					RpcMethod:      "RelayMiningDifficulty",
					Use:            "show-relay-mining-difficulty [id]",
					Short:          "Shows a relay-mining-difficulty",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				// {
				// 	RpcMethod: "UpdateParams",
				// 	Skip:      true, // skipped because authority gated
				// },
				// {
				// 	RpcMethod:      "UpdateParam",
				// 	Use:            "update-param [name] [as-type]",
				// 	Short:          "Send a update-param tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
