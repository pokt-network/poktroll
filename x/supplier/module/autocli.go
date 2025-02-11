package supplier

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/supplier"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Query_ServiceDesc.ServiceName,
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
<<<<<<< HEAD
					Use:       "list-suppliers [service-id]",
=======
					Use:       "list-suppliers",
>>>>>>> main
					Short:     "List all suppliers on Pocket Network",
					Long: `Retrieves a paginated list of all suppliers currently registered on Pocket Network, including all their details.

The command supports optional filtering by service ID and pagination parameters.
Returns supplier addresses, staked amounts, service details, and current status.`,

<<<<<<< HEAD
					Example: `
	poktrolld query supplier list-suppliers
=======
					Example: `	poktrolld query supplier list-suppliers
>>>>>>> main
	poktrolld query supplier list-suppliers --service-id anvil
	poktrolld query supplier list-suppliers --page 2 --limit 50
	poktrolld query supplier list-suppliers --service-id anvil --page 1 --limit 100`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"service_id": {Name: "service-id", Shorthand: "s", Usage: "service id to filter by", Hidden: false},
					},
<<<<<<< HEAD
					// PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}},
=======
>>>>>>> main
				},
				{
					Alias:     []string{"supplier", "s"},
					RpcMethod: "Supplier",
<<<<<<< HEAD
					Use:       "show-supplier [address]",
=======
					Use:       "show-supplier [operator_address]",
>>>>>>> main
					Short:     "Shows detailed information about a specific supplier",
					Long: `Retrieves comprehensive information about a supplier identified by their address.

Returns details include things like:
- Supplier's staked amount and status
- List of services they provide`,

<<<<<<< HEAD
					Example: `

	poktrolld query supplier show-supplier pokt1abc...xyz
=======
					Example: `	poktrolld query supplier show-supplier pokt1abc...xyz
>>>>>>> main
	poktrolld query supplier show-supplier pokt1abc...xyz --output json
	poktrolld query supplier show-supplier pokt1abc...xyz --height 100`,
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
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command (for backwards compatibility)
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
