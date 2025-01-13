package supplier

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/supplier"
)

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
					Use:       "list-supplier",
					Short:     "List all supplier",
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
