package gateway

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/gateway"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				//				{
				//					RpcMethod: "Params",
				//					Use:       "params",
				//					Short:     "Shows the parameters of the module",
				//				},
				//				{
				//					RpcMethod: "AllGateways",
				//					Use:       "list-gateway",
				//					Short:     "List all gateway",
				//				},
				//				{
				//					RpcMethod:      "Gateway",
				//					Use:            "show-gateway [id]",
				//					Short:          "Shows a gateway",
				//					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				//				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				//				{
				//					RpcMethod: "UpdateParams",
				//					Skip:      true, // skipped because authority gated
				//				},
				//				{
				//					RpcMethod:      "StakeGateway",
				//					Use:            "stake-gateway [stake]",
				//					Short:          "Send a stake_gateway tx",
				//					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}},
				//				},
				//				{
				//					RpcMethod:      "UnstakeGateway",
				//					Use:            "unstake-gateway",
				//					Short:          "Send a unstake_gateway tx",
				//					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				//				},
				{
					RpcMethod:      "UpdateParam",
					Use:            "update-param [name] [as-type]",
					Short:          "Send a update-param tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
