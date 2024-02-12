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
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "SupplierAll",
					Use:       "list-supplier",
					Short:     "List all supplier",
				},
				{
					RpcMethod:      "Supplier",
					Use:            "show-supplier [id]",
					Short:          "Shows a supplier",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
				{
					RpcMethod: "AllClaims",
					Use:       "list-claim",
					Short:     "List all claim",
				},
				{
					RpcMethod:      "Claim",
					Use:            "show-claim [id]",
					Short:          "Shows a claim",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
				{
					RpcMethod: "AllProofs",
					Use:       "list-proof",
					Short:     "List all proof",
				},
				{
					RpcMethod:      "Proof",
					Use:            "show-proof [id]",
					Short:          "Shows a proof",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
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
					RpcMethod:      "StakeSupplier",
					Use:            "stake-supplier [stake] [services]",
					Short:          "Send a stake-supplier tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
