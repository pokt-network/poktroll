package supplier

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/supplier"
)

// Query: &autocliv1.ServiceCommandDescriptor{
// 	Service: modulev1.Query_ServiceDesc.ServiceName,
// 	RpcCommandOptions: []*autocliv1.RpcCommandOptions{
// 		//			{
// 		//				RpcMethod: "Params",
// 		//				Use:       "params",
// 		//				Short:     "Shows the parameters of the module",
// 		//			},
// 		//			{
// 		//				RpcMethod: "AllServices",
// 		//				Use:       "list-service",
// 		//				Short:     "List all service",
// 		//			},
// 		{
// 			RpcMethod:      "Service",
// 			Use:            "show-service [id]",
// 			Short:          "Shows a service",
// 			Long:           "Retrieve the service details by its id.",
// 			PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
// 		},
// 		// this line is used by ignite scaffolding # autocli/query
// 	},
// },

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// {
				// 	RpcMethod: "Params",
				// 	Use:       "params",
				// 	Short:     "Shows the parameters of the module",
				// },
				{
					RpcMethod: "AllSuppliers",
					Use:       "list-suppliers",
					Short:     "List all suppliers on the Pocket Network",
					Long:      "Retrieves a paginated list of all suppliers currently registered on the Pocket Network, including their stakes and services.",
					Example:   "poktrolld query supplier list-suppliers\npoktrolld query supplier list-suppliers --page 2 --limit 50",
					Alias:     []string{"suppliers", "ls"},
				},
				// {
				// 	RpcMethod:      "Supplier",
				// 	Use:            "show-supplier [id]",
				// 	Short:          "Shows a supplier",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				// },
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				//{
				//	RpcMethod: "UpdateParams",
				//	Skip:      true, // skipped because authority gated
				//},
				//{
				//	RpcMethod:      "StakeSupplier",
				//	Use:            "stake-supplier [stake] [services]",
				//	Short:          "Send a stake-supplier tx",
				//	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				//},
				//{
				//	RpcMethod:      "UnstakeSupplier",
				//	Use:            "unstake-supplier",
				//	Short:          "Send a unstake-supplier tx",
				//	PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				//},
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
