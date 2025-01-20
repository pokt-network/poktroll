package migration

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/migration"
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
					RpcMethod: "MorseAccountState",
					Use:       "show-morse-account-state",
					Short:     "show morse_account_state",
					Skip:      true,
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
					RpcMethod:      "CreateMorseAccountState",
					Use:            "create-morse-account-state [accounts]",
					Short:          "Create morse_account_state",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "accounts"}},
					Skip:           true,
				},
				{
					RpcMethod:      "ClaimMorsePokt",
					Use:            "claim-morse-pokt [morse-src-address] [morse-signature]",
					Short:          "Send a claim-morse-pokt tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseSrcAddress"}, {ProtoField: "morseSignature"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
