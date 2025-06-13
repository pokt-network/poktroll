package supplier

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:              suppliertypes.Query_serviceDesc.ServiceName,
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
					Use:       "list-suppliers",
					Short:     "List all suppliers on Pocket Network",
					Long: `Retrieves a paginated list of all suppliers currently registered on Pocket Network, including all their details.

The command supports optional filtering by service ID and pagination parameters.
Returns supplier addresses, staked amounts, service details, and current status.`,

					Example: `	pocketd query supplier list-suppliers
	pocketd query supplier list-suppliers --service-id anvil
	pocketd query supplier list-suppliers --page 2 --limit 50
	pocketd query supplier list-suppliers --service-id anvil --page 1 --limit 100`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"service_id": {Name: "service-id", Shorthand: "s", Usage: "service id to filter by", Hidden: false},
					},
				},
				{
					Alias:     []string{"supplier", "s"},
					RpcMethod: "Supplier",
					Use:       "show-supplier [operator_address]",
					Short:     "Shows detailed information about a specific supplier",
					Long: `Retrieves comprehensive information about a supplier identified by their address.

Returns details include things like:
- Supplier's staked amount and status
- List of services they provide`,

					Example: `	pocketd query supplier show-supplier pokt1abc...xyz
	pocketd query supplier show-supplier pokt1abc...xyz --output json
	pocketd query supplier show-supplier pokt1abc...xyz --height 100`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "operator_address",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              suppliertypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command (for backwards compatibility)
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
				// {
				// 	RpcMethod:      "StakeSupplier",
				// 	Use:            "stake-supplier [stake] [services]",
				// 	Short:          "Send a stake-supplier tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				// },
				// {
				// 	RpcMethod:      "UnstakeSupplier",
				// 	Use:            "unstake-supplier",
				// 	Short:          "Send a unstake-supplier tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				// },
				// {
				// 	RpcMethod:      "UpdateParam",
				// 	Use:            "update-param [name] [as-type]",
				// 	Short:          "Send a update-param tx",
				// 	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
				// 	GovProposal: true,
				// 	// TODO_IN_THIS_COMMIT: update comment... preempt autoCLI for customization purposes.
				// 	Skip: true, // MUST be preempted by AddAutoCLICommands() in order to register custom flags.
				// },
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
