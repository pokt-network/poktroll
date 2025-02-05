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
				{
					RpcMethod: "MorseAccountClaimAll",
					Use:       "list-morse-account-claim",
					Short:     "List all morse_account_claim",
				},
				{
					RpcMethod: "MorseAccountClaim",
					Use:       "show-morse-account-claim --morse_src_address [morse_hex_address] | --shannon_dest_address [shannon_bech32_address]",
					Short:     "Shows a morse_account_claim by EITHER morse_src_address OR shannon_dest_address",
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
					RpcMethod:      "CreateMorseAccountClaim",
					Use:            "create-morse-account-claim [hex-morse-src-address] [hex-morse-signature]",
					Short:          "Create morse_account_claim",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morse_src_address"}, {ProtoField: "morse_signature"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
