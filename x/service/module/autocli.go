package service

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/pocket/service"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				//			{
				//				RpcMethod: "Params",
				//				Use:       "params",
				//				Short:     "Shows the parameters of the module",
				//			},
				//			{
				//				RpcMethod: "AllServices",
				//				Use:       "list-service",
				//				Short:     "List all service",
				//			},
				{
					RpcMethod:      "Service",
					Use:            "show-service [id]",
					Short:          "Shows a service",
					Long:           "Retrieve the service details by its id.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				//			{
				//				RpcMethod: "UpdateParams",
				//				Skip:      true, // skipped because authority gated
				//			},
				//			{
				//				RpcMethod:      "AddService",
				//				Use:            "add-service",
				//				Short:          "Send a add-service tx",
				//				PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				//			},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
