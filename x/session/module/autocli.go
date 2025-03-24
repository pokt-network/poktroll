package session

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/pocket/session"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:           modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// 				{
				// 					RpcMethod: "Params",
				// 					Use:       "params",
				// 					Short:     "Shows the parameters of the module",
				// 				},
				// 				{
				// 					RpcMethod: "GetSession",
				// 					Use:       "get-session [application-address] [service] [block-height]",
				// 					Short:     "Query get-session",
				// 					Long: `Query the session data for a specific (app, service, height) tuple.
				//
				// This is a query operation that will not result in a state transition but simply gives a view into the chain state.
				//
				// Example:
				// $ pocketd q session get-session pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 svc1 42 --node $(POCKET_NODE) --home $(POKTROLLD_HOME) `,
				// 					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "application_address"}, {ProtoField: "service"}, {ProtoField: "block_height"}},
				// 				},

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
