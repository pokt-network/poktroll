package migration

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: migrationtypes.Query_serviceDesc.ServiceName,
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
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              migrationtypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
				// TODO_IN_THIS_COMMIT: update comment about skipping beucause authority gated...
				// TODO_IN_THIS_COMMIT: update comment... explain that commenting is the new skipping,
				// and skipping is how we use AutoCLI with TX commands because we have to preempt it in order to register
				// custom flags. This means that we're creating the command, not autoCLI; therefore,
				// we need to skip it. We still use these conventional autoCLI data structures to
				// express the integration conventionally (save for the skips).
				// TODO_IN_THIS_COMMIT: consolidate existing custom commands with the commented ones.
				// Custom commands SHOULD be "justified"; i.e., AutoCLI integration is insufficient
				// for some reason. For example, a command is authority gated or requires non-trivial
				// custom logic like signature verification.
				// {
				// 	RpcMethod: "UpdateParams",
				// 	GovProposal: true,
				// 	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				// 	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				// },
				// 	RpcMethod:      "ImportMorseClaimableAccounts",
				// 	Use:            "import-morse-claimable-accounts [morse-account-state]",
				// 	Short:          "Send a import_morse_claimable_accounts tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseAccountState"}},
				// 	Skip:           true, // skipped because authority gated
				// },
				// {
				// 	RpcMethod:      "ClaimMorseAccount",
				// 	Use:            "claim-morse-account [morse-src-address-hex] [morse-signature-hex]",
				// 	Short:          "Claim the account balance of the given Morse account address",
				// 	Long:           "Claim the account balance of the given Morse account address, by signing the message with the private key of the Morse account.",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morse_src_address"}, {ProtoField: "morse_signature"}},
				// 	Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				// },
				// {
				// 	RpcMethod:      "ClaimMorseApplication",
				// 	Use:            "claim-morse-application [morse-src-address] [morse-signature] [stake] [service-config]",
				// 	Short:          "Send a claim_morse_application tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseSrcAddress"}, {ProtoField: "morseSignature"}, {ProtoField: "stake"}, {ProtoField: "serviceConfig"}},
				// 	Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				// },
				// {
				// 	RpcMethod:      "ClaimMorseSupplier",
				// 	Use:            "claim-morse-supplier [morse-src-address] [morse-signature] [stake] [service-config]",
				// 	Short:          "Send a claim_morse_supplier tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "morseSrcAddress"}, {ProtoField: "morseSignature"}, {ProtoField: "stake"}, {ProtoField: "serviceConfig"}},
				// 	Skip:           true, // skipped because autoCLI cannot handle loading & signing using the Morse key.
				// },
				// {
				// 	RpcMethod:      "RecoverMorseAccount",
				// 	Use:            "recover-morse-account [shannon-dest-address] [morse-src-address]",
				// 	Short:          "Send a recover_morse_account tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "shannonDestAddress"}, {ProtoField: "morseSrcAddress"}},
				// 	Skip:           true, // skipped because authority gated
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
