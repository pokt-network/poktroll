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
			Service:              modulev1.Query_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command (for backwards compatibility)
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// {
				// 	RpcMethod: "Params",
				// 	Use:       "params",
				// 	Short:     "Shows the parameters of the module",
				// },
				{
					Alias:     []string{"suppliers", "ls"},
					RpcMethod: "AllSuppliers",
					Use:       "list-suppliers [service-id]",
					Short:     "List all suppliers on Pocket Network",
					Long:      "Retrieves a paginated list of all suppliers currently registered on Pocket Network, including all their details.",
					Example:   " poktrolld query supplier list-suppliers \n poktrolld query supplier list-suppliers --service-id anvil \n poktrolld query supplier list-suppliers --page 2 --limit 50",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"service_id": {Name: "service-id", Shorthand: "s", Usage: "service id to filter by", Hidden: false},
					},
					// PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}},
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
			EnhanceCustomCommand: true, // only required if you want to use the custom command (for backwards compatibility)
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
