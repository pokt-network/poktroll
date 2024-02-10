package application

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/application"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ApplicationAll",
					Use:       "list-application",
					Short:     "List all application",
				},
				{
					RpcMethod:      "Application",
					Use:            "show-application [id]",
					Short:          "Shows a application",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "StakeApplication",
					Use:            "stake-application [stake] [services]",
					Short:          "Send a stake-application tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				},
				{
					RpcMethod:      "UnstakeApplication",
					Use:            "unstake-application",
					Short:          "Send a unstake-application tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "DelegateToGateway",
					Use:            "delegate-to-gateway [gateway-address]",
					Short:          "Send a delegate-to-gateway tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gatewayAddress"}},
				},
				{
					RpcMethod:      "UndelegateFromGateway",
					Use:            "undelegate-from-gateway [gateway-address]",
					Short:          "Send a undelegate-from-gateway tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gatewayAddress"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
