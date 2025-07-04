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
Returns supplier addresses, staked amounts, service details, and current status.

Use the --dehydrated flag to exclude service_config_history and rev_share details for more compact output.`,

					Example: `	pocketd query supplier list-suppliers
	pocketd query supplier list-suppliers --service-id anvil
	pocketd query supplier list-suppliers --dehydrated
	pocketd query supplier list-suppliers --page 2 --limit 50
	pocketd query supplier list-suppliers --service-id anvil --page 1 --limit 100
	pocketd query supplier list-suppliers --service-id anvil --dehydrated`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"service_id": {Name: "service-id", Shorthand: "s", Usage: "service id to filter by", Hidden: false},
						"dehydrated": {Name: "dehydrated", Shorthand: "d", Usage: "return suppliers with some fields omitted for a smaller response payload (e.g. service_config_history, rev_share, etc..)", Hidden: false},
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
- List of services they provide

Use the --dehydrated flag to exclude service_config_history and rev_share details for more compact output.`,

					Example: `	pocketd query supplier show-supplier pokt1abc...xyz
	pocketd query supplier show-supplier pokt1abc...xyz --output json
	pocketd query supplier show-supplier pokt1abc...xyz --height 100
	pocketd query supplier show-supplier pokt1abc...xyz --dehydrated`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "operator_address",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"dehydrated": {Name: "dehydrated", Shorthand: "d", Usage: "return supplier with some fields omitted for a smaller response payload (e.g. service_config_history, rev_share, etc..)", Hidden: false},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              suppliertypes.Msg_serviceDesc.ServiceName,
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
