package migration

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/pocket/migration"
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
					RpcMethod: "MorseClaimableAccountAll",
					Use:       "list-morse-claimable-account",
					Short:     "List all morse_claimable_account",
				},
				{
					RpcMethod:      "MorseClaimableAccount",
					Use:            "show-morse-claimable-account [id]",
					Short:          "Shows a morse_claimable_account",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod:      "MorseClaimableAccountsByShannonAddress",
					Use:            "morse-claimable-accounts-by-shannon-address [shannon-address]",
					Short:          "Query MorseClaimableAccountsByShannonAddress",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "shannonAddress"}},
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
					RpcMethod:      "ImportMorseClaimableAccounts",
					Use:            "import-morse-claimable-accounts [morse-account-state]",
					Short:          "Send a import_morse_claimable_accounts tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseAccountState"}},
					Skip:           true, // skipped because authority gated
				},
				{
					RpcMethod:      "ClaimMorseAccount",
					Use:            "claim-morse-account [morse-src-address-hex] [morse-signature-hex]",
					Short:          "Claim the account balance of the given Morse account address",
					Long:           "Claim the account balance of the given Morse account address, by signing the message with the private key of the Morse account.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morse_src_address"}, {ProtoField: "morse_signature"}},
					Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				},
				{
					RpcMethod:      "ClaimMorseApplication",
					Use:            "claim-morse-application [morse-src-address] [morse-signature] [stake] [service-config]",
					Short:          "Send a claim_morse_application tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseSrcAddress"}, {ProtoField: "morseSignature"}, {ProtoField: "stake"}, {ProtoField: "serviceConfig"}},
					Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				},
				{
					RpcMethod:      "ClaimMorseSupplier",
					Use:            "claim-morse-supplier [morse-src-address] [morse-signature] [stake] [service-config]",
					Short:          "Send a claim_morse_supplier tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseSrcAddress"}, {ProtoField: "morseSignature"}, {ProtoField: "stake"}, {ProtoField: "serviceConfig"}},
					Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				},
				{
					RpcMethod:      "RecoverMorseAccount",
					Use:            "recover-morse-account [shannon-dest-address] [morse-src-address]",
					Short:          "Send a recover_morse_account tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "shannonDestAddress"}, {ProtoField: "morseSrcAddress"}},
					Skip:           true, // skipped because authority gated
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
