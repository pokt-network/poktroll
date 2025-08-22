package shared

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: sharedtypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              sharedtypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "UpdateParam",
					Skip:      true, // skipped because authority gated
				},
				//	{
				//		RpcMethod:      "UpdateParam",
				//		Use:            "update-param [name] [as-type]",
				//		Short:          "Send a update-param tx",
				//		PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				//	},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
