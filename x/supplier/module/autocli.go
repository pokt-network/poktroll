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

The command supports optional filtering by service ID, operator address, owner address, and pagination parameters.
Multiple filters can be combined (AND logic).
Returns supplier addresses, staked amounts, service details, and current status.

Use the --dehydrated flag to exclude service_config_history and rev_share details for more compact output.`,

					Example: `	pocketd query supplier list-suppliers
	pocketd query supplier list-suppliers --service-id anvil
	pocketd query supplier list-suppliers --operator-address pokt1abc...xyz
	pocketd query supplier list-suppliers --owner-address pokt1owner...xyz
	pocketd query supplier list-suppliers --service-id anvil --operator-address pokt1abc...xyz
	pocketd query supplier list-suppliers --service-id anvil --owner-address pokt1owner...xyz
	pocketd query supplier list-suppliers --dehydrated
	pocketd query supplier list-suppliers --page 2 --limit 50
	pocketd query supplier list-suppliers --service-id anvil --page 1 --limit 100
	pocketd query supplier list-suppliers --service-id anvil --dehydrated`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"service_id":       {Name: "service-id", Shorthand: "s", Usage: "service id to filter by", Hidden: false},
						"operator_address": {Name: "operator-address", Usage: "operator address to filter by", Hidden: false},
						"owner_address":    {Name: "owner-address", Usage: "owner address to filter by", Hidden: false},
						"dehydrated":       {Name: "dehydrated", Shorthand: "d", Usage: "return suppliers with some fields omitted for a smaller response payload (e.g. service_config_history, rev_share, etc..)", Hidden: false},
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
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "UpdateParam",
					Skip:      true, // skipped because authority gated
				},
				//{
				//	RpcMethod:      "StakeSupplier",
				//	Use:            "stake-supplier [stake] [services]",
				//	Short:          "Send a stake-supplier tx",
				//	PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "stake"}, {ProtoField: "services"}},
				//},
				{
					RpcMethod: "UnstakeSupplier",
					Use:       "unstake-supplier [operator_address]",
					Short:     "Unstake a supplier from Pocket Network",
					Long: `Unstake a supplier with the provided operator address.

This is an onchain transaction that will initiate the unstaking process for the supplier.

The --from flag specifies the signer and can be either:
- The supplier owner address (who originally staked the supplier)
- The operator address (the service provider address)

The [operator_address] argument is the operator address of the supplier to unstake.
The staked funds will always be returned to the owner address, regardless of who initiates the unstaking.

The supplier will continue providing service until the current session ends, at which point it will be deactivated.`,

					Example: `
	# Unstake supplier as owner
	pocketd tx supplier unstake-supplier pokt1operator... --from pokt1owner... --keyring-backend test --network mainnet

	# Unstake supplier as operator
	pocketd tx supplier unstake-supplier pokt1operator... --from pokt1operator... --keyring-backend test --network mainnet

	# With custom home directory
	pocketd tx supplier unstake-supplier pokt1operator... --from pokt1owner... --home ./pocket --keyring-backend test --network mainnet`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "operator_address",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
